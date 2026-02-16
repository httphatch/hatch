package process

import (
	"context"
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
	Command  string    `json:"command"`
	Running  bool      `json:"running"`
	Restarts int       `json:"restarts"`
	StartedAt time.Time `json:"started_at"`
}

// ManagerConfig holds the configuration for a process Manager.
type ManagerConfig struct {
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

	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop processes no longer in config.
	for id, sup := range m.processes {
		if _, ok := desired[id]; !ok {
			sup.cancel()
			<-sup.done
			delete(m.processes, id)
		}
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
			<-existing.done
			delete(m.processes, id)
		}

		sup := m.startSupervised(id, want.command, want.dir, want.env)
		m.processes[id] = sup
	}

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
		<-sup.done
	}

	m.mu.Lock()
	m.processes = make(map[ServiceID]*supervised)
	m.mu.Unlock()

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
			Restarts:  sup.restarts,
			StartedAt: sup.started,
		}
		sup.mu.Unlock()
	}
	return out
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

			desired[id] = desiredProcess{
				command: svc.Command,
				dir:     proj.Path,
				env:     env,
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
		started: time.Now(),
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

		sup.mu.Lock()
		sup.runner = runner
		sup.mu.Unlock()

		if err := runner.Start(ctx); err != nil {
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

		startedAt := time.Now()

		// Wait for exit or cancellation.
		select {
		case <-ctx.Done():
			_ = runner.Stop()
			return
		case <-runner.Done():
		}

		// Don't restart if context is cancelled.
		if ctx.Err() != nil {
			return
		}

		// Reset backoff if the process ran long enough.
		if time.Since(startedAt) >= backoffResetTime {
			backoff = minBackoff
		}

		sup.mu.Lock()
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

func nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > maxBackoff {
		return maxBackoff
	}
	return next
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
