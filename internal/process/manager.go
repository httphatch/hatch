package process

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/httphatch/hatch/internal/config"
)

const (
	minBackoff       = 1 * time.Second
	maxBackoff       = 30 * time.Second
	backoffResetTime = 60 * time.Second
)

var (
	ErrProcessNotFound   = errors.New("process not found")
	ErrAlreadyStopped    = errors.New("process is already stopped")
	ErrAlreadyRunning    = errors.New("process is already running")
)

// ServiceID uniquely identifies a managed process.
type ServiceID struct {
	Project string
	Service string
}

func (id ServiceID) String() string {
	return id.Project + "/" + id.Service
}

// ProcessStatus holds the current state of a managed process.
type ProcessStatus struct {
	Command   string    `json:"command"`
	Running   bool      `json:"running"`
	Stopped   bool      `json:"stopped"`
	Restarts  int       `json:"restarts"`
	StartedAt time.Time `json:"started_at"`
}

// ManagerConfig holds the configuration for a process Manager.
type ManagerConfig struct {
	PIDFile  string
	OnOutput func(project, service, source, line string)
}

// supervised holds the state for a single supervised process.
type supervised struct {
	id      ServiceID
	command string
	dir     string
	env     []string
	runner  *Runner
	cancel  context.CancelFunc
	done    chan struct{} // closed when supervision goroutine exits

	mu       sync.Mutex
	restarts int
	started  time.Time
	stopped  bool
}

// Manager supervises processes defined in the Hatch config.
type Manager struct {
	cfg       ManagerConfig
	mu        sync.Mutex
	processes map[ServiceID]*supervised
}

// NewManager creates a new process Manager.
func NewManager(cfg ManagerConfig) *Manager {
	return &Manager{
		cfg:       cfg,
		processes: make(map[ServiceID]*supervised),
	}
}

// ApplyConfig reconciles running processes with the given config.
// It starts new processes, stops removed ones, and restarts changed ones.
func (m *Manager) ApplyConfig(appCfg config.Config) error {
	desired := m.buildDesired(appCfg)

	// Stop processes no longer in config. Release the lock before waiting
	// so Statuses() and Stop() are not blocked during graceful shutdown.
	m.mu.Lock()
	var removed []*supervised
	var removedIDs []ServiceID
	for id, sup := range m.processes {
		if _, ok := desired[id]; !ok {
			sup.cancel()
			removed = append(removed, sup)
			removedIDs = append(removedIDs, id)
		}
	}
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, sup := range removed {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-sup.done
		}()
	}
	wg.Wait()

	m.mu.Lock()
	for _, id := range removedIDs {
		delete(m.processes, id)
	}

	// Start or restart processes.
	for id, want := range desired {
		if existing, ok := m.processes[id]; ok {
			// Check if config changed.
			if existing.command == want.command && existing.dir == want.dir && envEqual(existing.env, want.env) {
				continue
			}
			// Restart: stop old, start new.
			existing.cancel()
			m.mu.Unlock()
			<-existing.done
			m.mu.Lock()
			delete(m.processes, id)
		}

		sup := m.startSupervised(id, want.command, want.dir, want.env)
		m.processes[id] = sup
	}
	m.mu.Unlock()

	return nil
}

// Stop terminates all managed processes.
func (m *Manager) Stop() error {
	m.mu.Lock()
	procs := make(map[ServiceID]*supervised, len(m.processes))
	for k, v := range m.processes {
		procs[k] = v
	}
	m.mu.Unlock()

	for _, sup := range procs {
		sup.cancel()
	}
	var wg sync.WaitGroup
	for _, sup := range procs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-sup.done
		}()
	}
	wg.Wait()

	m.mu.Lock()
	m.processes = make(map[ServiceID]*supervised)
	m.mu.Unlock()

	if m.cfg.PIDFile != "" {
		_ = os.Remove(m.cfg.PIDFile)
	}

	return nil
}

// Statuses returns a snapshot of all managed process statuses.
func (m *Manager) Statuses() map[ServiceID]ProcessStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make(map[ServiceID]ProcessStatus, len(m.processes))
	for id, sup := range m.processes {
		sup.mu.Lock()
		running := false
		if sup.runner != nil {
			select {
			case <-sup.runner.Done():
			default:
				running = true
			}
		}
		out[id] = ProcessStatus{
			Command:   sup.command,
			Running:   running,
			Stopped:   sup.stopped,
			Restarts:  sup.restarts,
			StartedAt: sup.started,
		}
		sup.mu.Unlock()
	}
	return out
}

// StopProcess manually stops a running process. The process remains in the
// process map but will not auto-restart until explicitly started again.
func (m *Manager) StopProcess(id ServiceID) error {
	m.mu.Lock()
	sup, ok := m.processes[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("%w: %v", ErrProcessNotFound, id)
	}
	sup.mu.Lock()
	if sup.stopped {
		sup.mu.Unlock()
		m.mu.Unlock()
		return fmt.Errorf("%w: %v", ErrAlreadyStopped, id)
	}
	sup.stopped = true
	sup.mu.Unlock()
	sup.cancel()
	m.mu.Unlock()
	<-sup.done
	return nil
}

// StartProcess resumes a manually stopped process by replacing the old
// supervised entry with a fresh one, avoiding mutation of shared fields.
func (m *Manager) StartProcess(id ServiceID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sup, ok := m.processes[id]
	if !ok {
		return fmt.Errorf("%w: %v", ErrProcessNotFound, id)
	}
	sup.mu.Lock()
	if !sup.stopped {
		sup.mu.Unlock()
		return fmt.Errorf("%w: %v", ErrAlreadyRunning, id)
	}
	sup.mu.Unlock()

	newSup := m.startSupervised(id, sup.command, sup.dir, sup.env)
	m.processes[id] = newSup
	return nil
}

// RestartProcess restarts a process. If running, it stops then starts it.
// If already stopped, it just starts it.
func (m *Manager) RestartProcess(id ServiceID) error {
	m.mu.Lock()
	sup, ok := m.processes[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("%w: %v", ErrProcessNotFound, id)
	}
	sup.mu.Lock()
	wasStopped := sup.stopped
	sup.mu.Unlock()
	m.mu.Unlock()

	if !wasStopped {
		if err := m.StopProcess(id); err != nil {
			return fmt.Errorf("stopping process: %w", err)
		}
	}
	return m.StartProcess(id)
}

type desiredProcess struct {
	command string
	dir     string
	env     []string
}

func (m *Manager) buildDesired(appCfg config.Config) map[ServiceID]desiredProcess {
	desired := make(map[ServiceID]desiredProcess)
	for projName, proj := range appCfg.Projects {
		if !proj.Enabled {
			continue
		}
		if proj.Path != "" {
			if _, err := os.Stat(proj.Path); err != nil {
				log.Warn().
					Str("project", projName).
					Str("path", proj.Path).
					Msg("project path does not exist, skipping managed processes")
				continue
			}
		}
		for svcName, svc := range proj.Services {
			if svc.Command == "" {
				continue
			}

			id := ServiceID{Project: projName, Service: svcName}
			env := os.Environ()

			// Load env file if specified.
			if svc.EnvFile != "" {
				envFilePath := svc.EnvFile
				if !filepath.IsAbs(envFilePath) && proj.Path != "" {
					envFilePath = filepath.Join(proj.Path, envFilePath)
				}
				envFilePath = filepath.Clean(envFilePath)

				// Block path traversal outside the project directory.
				if proj.Path != "" && !strings.HasPrefix(envFilePath, filepath.Clean(proj.Path)+string(filepath.Separator)) {
					log.Warn().
						Str("project", projName).
						Str("service", svcName).
						Str("env_file", svc.EnvFile).
						Msg("env_file path escapes project directory, skipping")
				} else if fileEnv, err := LoadEnvFile(envFilePath); err == nil {
					for k, v := range fileEnv {
						env = append(env, k+"="+v)
					}
				} else {
					log.Warn().Err(err).
						Str("project", projName).
						Str("service", svcName).
						Str("env_file", envFilePath).
						Msg("failed to load env file")
				}
			}

			dir := proj.Path
			if svc.Dir != "" {
				dir = filepath.Clean(filepath.Join(proj.Path, svc.Dir))
				if proj.Path != "" && !strings.HasPrefix(dir, filepath.Clean(proj.Path)+string(filepath.Separator)) {
					log.Warn().
						Str("project", projName).
						Str("service", svcName).
						Str("dir", svc.Dir).
						Msg("dir path escapes project directory, using project path")
					dir = proj.Path
				}
			}

			desired[id] = desiredProcess{
				command: svc.Command,
				dir:     dir,
				env:     deduplicateEnv(env),
			}
		}
	}
	return desired
}

func (m *Manager) startSupervised(id ServiceID, command, dir string, env []string) *supervised {
	ctx, cancel := context.WithCancel(context.Background())
	sup := &supervised{
		id:      id,
		command: command,
		dir:     dir,
		env:     env,
		cancel:  cancel,
		done:    make(chan struct{}),
	}

	go m.supervise(ctx, sup)

	return sup
}

func (m *Manager) supervise(ctx context.Context, sup *supervised) {
	defer close(sup.done)

	backoff := minBackoff

	for {
		runner := &Runner{cfg: RunnerConfig{
			Name:    sup.id.String(),
			Command: sup.command,
			Dir:     sup.dir,
			Env:     sup.env,
			OnOutput: func(name, source, line string) {
				if m.cfg.OnOutput != nil {
					m.cfg.OnOutput(sup.id.Project, sup.id.Service, source, line)
				}
			},
		}}

		if err := runner.Start(); err != nil {
			sup.mu.Lock()
			sup.runner = nil
			sup.mu.Unlock()

			// If context was cancelled, exit supervision.
			if ctx.Err() != nil {
				return
			}
			// Wait before retrying on start failure.
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				backoff = nextBackoff(backoff)
				continue
			}
		}

		now := time.Now()
		sup.mu.Lock()
		sup.runner = runner
		sup.started = now
		sup.mu.Unlock()

		m.mu.Lock()
		m.savePIDs()
		m.mu.Unlock()

		startedAt := now

		// Wait for exit or cancellation.
		select {
		case <-ctx.Done():
			_ = runner.Stop()
			return
		case <-runner.Done():
		}

		m.mu.Lock()
		m.savePIDs()
		m.mu.Unlock()

		// Don't restart if context is cancelled or manually stopped.
		if ctx.Err() != nil {
			return
		}

		sup.mu.Lock()
		if sup.stopped {
			sup.mu.Unlock()
			return
		}
		if time.Since(startedAt) >= backoffResetTime {
			backoff = minBackoff
			sup.restarts = 0
		}
		sup.restarts++
		sup.mu.Unlock()

		// Wait before restarting.
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			backoff = nextBackoff(backoff)
		}
	}
}

// savePIDs writes the current process PIDs to the state file.
// Must be called with m.mu held or from a context where the map is stable.
func (m *Manager) savePIDs() {
	if m.cfg.PIDFile == "" {
		return
	}

	pids := make(map[string]int)
	for id, sup := range m.processes {
		sup.mu.Lock()
		if sup.runner != nil {
			if pid := sup.runner.PID(); pid != 0 {
				pids[id.String()] = pid
			}
		}
		sup.mu.Unlock()
	}

	data, err := json.Marshal(pids)
	if err != nil {
		log.Warn().Err(err).Msg("failed to marshal process PIDs")
		return
	}

	if err := os.WriteFile(m.cfg.PIDFile, data, 0o600); err != nil {
		log.Warn().Err(err).Msg("failed to write process PID file")
	}
}

func nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > maxBackoff {
		return maxBackoff
	}
	return next
}

// deduplicateEnv keeps the last occurrence of each environment variable key.
// This ensures env_file values override inherited environment variables.
func deduplicateEnv(env []string) []string {
	seen := make(map[string]int, len(env))
	for i, entry := range env {
		if k, _, ok := strings.Cut(entry, "="); ok {
			seen[k] = i
		}
	}
	result := make([]string, 0, len(seen))
	for i, entry := range env {
		if k, _, ok := strings.Cut(entry, "="); ok {
			if seen[k] == i {
				result = append(result, entry)
			}
		}
	}
	return result
}

func envEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]bool, len(a))
	for _, v := range a {
		set[v] = true
	}
	for _, v := range b {
		if !set[v] {
			return false
		}
	}
	return true
}
