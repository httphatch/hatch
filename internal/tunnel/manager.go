package tunnel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/httphatch/hatch/internal/cloudflare"
	"github.com/httphatch/hatch/internal/config"
)

// TunnelID uniquely identifies a tunnel.
type TunnelID struct {
	Project string
	Service string
}

func (id TunnelID) String() string {
	return id.Project + "/" + id.Service
}

// TunnelStatus holds the current state of a tunnel.
type TunnelStatus struct {
	Running   bool      `json:"running"`
	Starting  bool      `json:"starting"`
	URL       string    `json:"url"`
	Type      string    `json:"type"` // "quick" or "named"
	StartedAt time.Time `json:"started_at"`
	Error     string    `json:"error,omitempty"`
}

type managed struct {
	runner *Runner
	status TunnelStatus
	done   <-chan struct{}
}

// TokenResolver resolves a tunnel JWT from a Cloudflare API token.
type TokenResolver interface {
	ResolveTunnelToken(apiToken, accountID, tunnelName string) (string, error)
}

// Manager supervises cloudflared tunnel processes.
type Manager struct {
	mu            sync.Mutex
	tunnels       map[TunnelID]*managed
	stateFile     string
	onOutput      func(project, service, source, line string)
	tokenResolver TokenResolver
}

// NewManager creates a new tunnel Manager.
// stateFile is the path to persist tunnel state (e.g. ~/.hatch/tunnels.json).
// onOutput, if non-nil, receives cloudflared output lines for logging and streaming.
func NewManager(stateFile string, onOutput func(project, service, source, line string)) *Manager {
	return &Manager{
		tunnels:       make(map[TunnelID]*managed),
		stateFile:     stateFile,
		onOutput:      onOutput,
		tokenResolver: cloudflare.NewClient(),
	}
}

// StartTunnel starts a tunnel for the given service.
func (m *Manager) StartTunnel(id TunnelID, upstream, tunnelName, token, accountID string) error {
	m.mu.Lock()
	if t, ok := m.tunnels[id]; ok && (t.status.Running || t.status.Starting) {
		m.mu.Unlock()
		if t.status.Starting {
			return fmt.Errorf("tunnel already starting: %s", id)
		}
		return fmt.Errorf("tunnel already running: %s", id)
	}
	m.mu.Unlock()

	if upstream == "" {
		return fmt.Errorf("service has no upstream to tunnel")
	}

	tunnelType := "quick"
	if tunnelName != "" && tunnelName != "true" {
		tunnelType = "named"
	} else {
		tunnelName = ""
	}

	// For named tunnels with an API token, resolve the tunnel JWT.
	// Without an API token, cloudflared falls back to credential files.
	if tunnelType == "named" && token != "" {
		jwt, err := m.tokenResolver.ResolveTunnelToken(token, accountID, tunnelName)
		if err != nil {
			return fmt.Errorf("resolving tunnel token for %s: %w", id, err)
		}
		token = jwt
	}

	cfPath, err := FindCloudflared()
	if err != nil {
		return err
	}

	// Register as "starting" so the dashboard can show a loading state
	// while cloudflared is initializing (can take up to 30s for quick tunnels).
	mgd := &managed{
		status: TunnelStatus{
			Starting: true,
			Type:     tunnelType,
		},
	}
	m.mu.Lock()
	m.tunnels[id] = mgd
	m.mu.Unlock()

	runner := &Runner{cfg: RunnerConfig{
		Upstream:        upstream,
		TunnelName:      tunnelName,
		Token:           token,
		CloudflaredPath: cfPath,
		OnOutput: func(source, line string) {
			log.Debug().
				Str("tunnel", id.String()).
				Str("source", source).
				Msg(line)
			if m.onOutput != nil {
				m.onOutput(id.Project, id.Service, source, line)
			}
		},
	}}

	if err := runner.Start(); err != nil {
		m.mu.Lock()
		delete(m.tunnels, id)
		m.mu.Unlock()
		return fmt.Errorf("starting tunnel %s: %w", id, err)
	}

	m.mu.Lock()
	mgd.runner = runner
	mgd.status = TunnelStatus{
		Running:   true,
		URL:       runner.URL(),
		Type:      tunnelType,
		StartedAt: time.Now(),
	}
	mgd.done = runner.Done()
	m.mu.Unlock()

	m.saveState()

	// Monitor for unexpected exit.
	go func() {
		<-mgd.done
		m.mu.Lock()
		if current, ok := m.tunnels[id]; ok && current == mgd {
			current.status.Running = false
			current.status.Error = "tunnel process exited"
		}
		m.mu.Unlock()
		m.saveState()
	}()

	return nil
}

// StopTunnel stops a running tunnel.
func (m *Manager) StopTunnel(id TunnelID) error {
	m.mu.Lock()
	mgd, ok := m.tunnels[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("tunnel not found: %s", id)
	}
	if !mgd.status.Running {
		m.mu.Unlock()
		return fmt.Errorf("tunnel already stopped: %s", id)
	}
	m.mu.Unlock()

	if err := mgd.runner.Stop(); err != nil {
		return fmt.Errorf("stopping tunnel %s: %w", id, err)
	}

	m.mu.Lock()
	delete(m.tunnels, id)
	m.mu.Unlock()

	m.saveState()
	return nil
}

// Statuses returns a snapshot of all tunnel statuses.
func (m *Manager) Statuses() map[TunnelID]TunnelStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make(map[TunnelID]TunnelStatus, len(m.tunnels))
	for id, mgd := range m.tunnels {
		out[id] = mgd.status
	}
	return out
}

// StopAll stops all running tunnels.
func (m *Manager) StopAll() error {
	m.mu.Lock()
	tunnels := make(map[TunnelID]*managed, len(m.tunnels))
	for k, v := range m.tunnels {
		tunnels[k] = v
	}
	m.mu.Unlock()

	for _, mgd := range tunnels {
		if mgd.status.Running {
			_ = mgd.runner.Stop()
		}
	}

	m.mu.Lock()
	m.tunnels = make(map[TunnelID]*managed)
	m.mu.Unlock()

	m.saveState()
	return nil
}

// ApplyConfig reconciles running tunnels with the given config.
func (m *Manager) ApplyConfig(cfg config.Config) error {
	desired := m.buildDesired(cfg)

	// Stop tunnels no longer in config.
	m.mu.Lock()
	var toStop []TunnelID
	for id := range m.tunnels {
		if _, ok := desired[id]; !ok {
			toStop = append(toStop, id)
		}
	}
	m.mu.Unlock()

	for _, id := range toStop {
		if err := m.StopTunnel(id); err != nil {
			log.Warn().Err(err).Str("tunnel", id.String()).Msg("failed to stop removed tunnel")
		}
	}

	// Start new tunnels in the background so they don't block
	// the daemon boot sequence (quick tunnels can take up to 30s
	// to obtain a URL from cloudflared).
	for id, want := range desired {
		m.mu.Lock()
		_, exists := m.tunnels[id]
		m.mu.Unlock()

		if exists {
			continue
		}

		go func(id TunnelID, want desiredTunnel) {
			if err := m.StartTunnel(id, want.upstream, want.tunnelName, want.token, want.accountID); err != nil {
				log.Warn().Err(err).Str("tunnel", id.String()).Msg("failed to start tunnel")
			}
		}(id, want)
	}

	return nil
}

type desiredTunnel struct {
	upstream   string
	tunnelName string
	token      string
	accountID  string
}

func (m *Manager) buildDesired(cfg config.Config) map[TunnelID]desiredTunnel {
	desired := make(map[TunnelID]desiredTunnel)
	for projName, proj := range cfg.Projects {
		if !proj.Enabled {
			continue
		}
		for svcName, svc := range proj.Services {
			tunnelVal := string(svc.Tunnel)
			if tunnelVal == "" {
				continue
			}
			if svc.Proxy == "" {
				continue
			}

			token := proj.CloudflareToken
			if token == "" {
				token = cfg.Settings.CloudflareToken
			}

			desired[TunnelID{Project: projName, Service: svcName}] = desiredTunnel{
				upstream:   svc.Proxy,
				tunnelName: tunnelVal,
				token:      token,
				accountID:  cfg.Settings.CloudflareAccountID,
			}
		}
	}
	return desired
}

type persistedTunnel struct {
	URL       string `json:"url"`
	Type      string `json:"type"`
	StartedAt string `json:"started_at"`
}

// saveState persists running tunnel info to disk for CLI status display.
// This is informational only; the manager does not restore state on startup.
func (m *Manager) saveState() {
	m.mu.Lock()
	state := make(map[string]persistedTunnel)
	for id, mgd := range m.tunnels {
		if mgd.status.Running {
			state[id.String()] = persistedTunnel{
				URL:       mgd.status.URL,
				Type:      mgd.status.Type,
				StartedAt: mgd.status.StartedAt.Format(time.RFC3339),
			}
		}
	}
	m.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.Warn().Err(err).Msg("failed to marshal tunnel state")
		return
	}

	dir := filepath.Dir(m.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Warn().Err(err).Msg("failed to create tunnel state directory")
		return
	}

	if err := os.WriteFile(m.stateFile, data, 0600); err != nil {
		log.Warn().Err(err).Msg("failed to write tunnel state")
	}
}
