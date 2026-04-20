package session

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/httphatch/hatch/internal/config"
)

var validSessionName = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

const (
	DefaultTTL         = 30 * time.Minute
	DefaultMaxSessions = 20
	autoNamePrefix     = "s"
)

var (
	ErrProjectNotFound  = fmt.Errorf("project not found")
	ErrProjectDisabled  = fmt.Errorf("project is disabled")
	ErrSessionExists    = fmt.Errorf("session already exists")
	ErrSessionNotFound  = fmt.Errorf("session not found")
	ErrNoSessionServices = fmt.Errorf("project has no services with a command field")
	ErrNameConflict     = fmt.Errorf("session name conflicts with an existing subdomain")
	ErrMaxSessions      = fmt.Errorf("maximum number of sessions reached")
)

// ManagerConfig holds the configuration for a session Manager.
type ManagerConfig struct {
	StateFile  string
	PortMin    int
	PortMax    int
	DefaultTTL time.Duration
	OnReload   func() // triggers Caddy config + process reload
}

// Manager coordinates session lifecycle: create, destroy, merge, reconcile.
type Manager struct {
	mu       sync.Mutex
	sessions map[SessionID]*Session
	ports    *PortAllocator
	cfg      ManagerConfig
	counter  int // for auto-naming
}

// NewManager creates a session manager.
func NewManager(cfg ManagerConfig) *Manager {
	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = DefaultTTL
	}
	return &Manager{
		sessions: make(map[SessionID]*Session),
		ports:    NewPortAllocator(cfg.PortMin, cfg.PortMax),
		cfg:      cfg,
	}
}

// Ports returns the underlying port allocator.
func (m *Manager) Ports() *PortAllocator {
	return m.ports
}

// CreateSession allocates ports, registers the session, and triggers a reload.
func (m *Manager) CreateSession(project, name string, ttl time.Duration, baseCfg config.Config) (*Session, error) {
	proj, ok := baseCfg.Projects[project]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProjectNotFound, project)
	}
	if !proj.Enabled {
		return nil, fmt.Errorf("%w: %s", ErrProjectDisabled, project)
	}

	// Collect services that have a command (session-eligible), sorted for deterministic port assignment.
	var serviceNames []string
	for svcName, svc := range proj.Services {
		if svc.Command != "" {
			serviceNames = append(serviceNames, svcName)
		}
	}
	if len(serviceNames) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoSessionServices, project)
	}
	sort.Strings(serviceNames)

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) >= DefaultMaxSessions {
		return nil, fmt.Errorf("%w: limit is %d", ErrMaxSessions, DefaultMaxSessions)
	}

	// Auto-generate name if empty.
	if name == "" {
		name = m.generateName(project)
	}

	// Validate name is a valid hostname label (it becomes a subdomain).
	if !validSessionName.MatchString(name) {
		return nil, fmt.Errorf("invalid session name %q: must be a valid hostname label (alphanumeric and hyphens, 1-63 chars)", name)
	}

	id := SessionID{Project: project, Name: name}

	if _, exists := m.sessions[id]; exists {
		return nil, fmt.Errorf("%w: %s", ErrSessionExists, id)
	}

	// Validate name doesn't conflict with existing subdomains.
	if err := m.validateName(name, proj); err != nil {
		return nil, err
	}

	// Allocate ports.
	ports, err := m.ports.Allocate(id, len(serviceNames))
	if err != nil {
		return nil, fmt.Errorf("allocating ports: %w", err)
	}

	portMap := make(map[string]int, len(serviceNames))
	for i, svcName := range serviceNames {
		portMap[svcName] = ports[i]
	}

	// Build domain map.
	domains := make(map[string]string, len(serviceNames))
	for _, svcName := range serviceNames {
		svc := proj.Services[svcName]
		domains[svcName] = Domain(name, proj.Domain, svc.Subdomain)
	}

	if ttl < 0 {
		return nil, fmt.Errorf("ttl must be >= 0")
	}
	if ttl == 0 {
		ttl = m.cfg.DefaultTTL
	}

	now := time.Now()
	sess := &Session{
		ID:         id,
		State:      StateRunning,
		BaseDomain: proj.Domain,
		Ports:      portMap,
		Domains:    domains,
		CreatedAt:  now,
		LastActive: now,
		TTL:        ttl,
	}
	m.sessions[id] = sess

	if err := saveState(m.cfg.StateFile, m.sessions); err != nil {
		log.Warn().Err(err).Msg("failed to save session state")
	}

	// Trigger reload outside the lock to avoid deadlock.
	go m.cfg.OnReload()

	return sess, nil
}

// DestroySession stops a session, releases its ports, and triggers a reload.
func (m *Manager) DestroySession(id SessionID) error {
	m.mu.Lock()
	sess, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrSessionNotFound, id)
	}

	sess.State = StateStopped
	m.ports.Release(id)
	delete(m.sessions, id)

	if err := saveState(m.cfg.StateFile, m.sessions); err != nil {
		log.Warn().Err(err).Msg("failed to save session state")
	}
	m.mu.Unlock()

	m.cfg.OnReload()
	return nil
}

// Sessions returns a snapshot of all sessions.
func (m *Manager) Sessions() []Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, *s)
	}
	return out
}

// GetSession returns a copy of a specific session.
func (m *Manager) GetSession(id SessionID) (Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[id]
	if !ok {
		return Session{}, false
	}
	return *s, true
}

// RecordActivity updates LastActive for the session matching the given hostname.
func (m *Manager) RecordActivity(hostname string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, sess := range m.sessions {
		for _, domain := range sess.Domains {
			if domain == hostname {
				sess.LastActive = time.Now()
				return
			}
		}
	}
}

// MergeIntoConfig produces a config with synthetic session projects appended.
// The synthetic projects use the session's allocated ports as proxy targets.
func (m *Manager) MergeIntoConfig(baseCfg config.Config) config.Config {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) == 0 {
		return baseCfg
	}

	// Shallow copy projects map.
	merged := baseCfg
	merged.Projects = make(map[string]config.Project, len(baseCfg.Projects)+len(m.sessions))
	for k, v := range baseCfg.Projects {
		merged.Projects[k] = v
	}

	for _, sess := range m.sessions {
		if sess.State != StateRunning {
			continue
		}
		baseProj, ok := baseCfg.Projects[sess.ID.Project]
		if !ok {
			continue
		}

		syntheticServices := make(map[string]config.Service, len(sess.Ports))
		for svcName, port := range sess.Ports {
			baseSvc := baseProj.Services[svcName]
			syntheticServices[svcName] = config.Service{
				Proxy:     fmt.Sprintf("http://localhost:%d", port),
				Command:   SubstituteCommand(baseSvc.Command, svcName, sess.Ports),
				Dir:       baseSvc.Dir,
				Route:     baseSvc.Route,
				Subdomain: baseSvc.Subdomain,
				WebSocket: baseSvc.WebSocket,
				EnvFile:   baseSvc.EnvFile,
			}
		}

		merged.Projects[sess.ID.QualifiedProject()] = config.Project{
			Domain:   Domain(sess.ID.Name, sess.BaseDomain, ""),
			Path:     baseProj.Path,
			Enabled:  true,
			Services: syntheticServices,
		}
	}

	return merged
}

// ReconcileWithConfig destroys sessions whose parent project is disabled or removed.
func (m *Manager) ReconcileWithConfig(cfg config.Config) {
	m.mu.Lock()
	var toDestroy []SessionID
	for id := range m.sessions {
		proj, exists := cfg.Projects[id.Project]
		if !exists || !proj.Enabled {
			toDestroy = append(toDestroy, id)
		}
	}
	m.mu.Unlock()

	for _, id := range toDestroy {
		log.Info().Str("session", id.String()).Msg("destroying session (parent project disabled or removed)")
		if err := m.DestroySession(id); err != nil {
			log.Warn().Err(err).Str("session", id.String()).Msg("failed to destroy session")
		}
	}
}

// RestoreState re-creates sessions from the state file.
func (m *Manager) RestoreState(baseCfg config.Config) error {
	persisted, err := loadState(m.cfg.StateFile)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range persisted {
		proj, ok := baseCfg.Projects[p.Project]
		if !ok || !proj.Enabled {
			continue
		}

		id := SessionID{Project: p.Project, Name: p.Name}

		// Validate session name from state file.
		if !validSessionName.MatchString(p.Name) {
			log.Warn().Str("session", id.String()).Msg("invalid session name in state file, skipping")
			continue
		}

		// Try to reclaim the same ports.
		portSlice := make([]int, 0, len(p.Ports))
		for _, port := range p.Ports {
			portSlice = append(portSlice, port)
		}
		if err := m.ports.AllocateSpecific(id, portSlice); err != nil {
			// Ports taken, allocate new ones.
			log.Warn().Err(err).Str("session", id.String()).Msg("original ports unavailable, allocating new ones")
			newPorts, err := m.ports.Allocate(id, len(p.Ports))
			if err != nil {
				log.Warn().Err(err).Str("session", id.String()).Msg("failed to reallocate ports, skipping session")
				continue
			}
			// Sort service names for deterministic port assignment.
			svcNames := sortedKeys(p.Ports)
			for i, svcName := range svcNames {
				p.Ports[svcName] = newPorts[i]
			}
		}

		// Rebuild domains.
		domains := make(map[string]string, len(p.Ports))
		for svcName := range p.Ports {
			svc := proj.Services[svcName]
			domains[svcName] = Domain(p.Name, proj.Domain, svc.Subdomain)
		}

		ttl := time.Duration(p.TTLSeconds) * time.Second
		if ttl == 0 {
			ttl = m.cfg.DefaultTTL
		}

		m.sessions[id] = &Session{
			ID:         id,
			State:      StateRunning,
			BaseDomain: proj.Domain,
			Ports:      p.Ports,
			Domains:    domains,
			CreatedAt:  p.CreatedAt,
			LastActive: time.Now(), // reset on restore
			TTL:        ttl,
		}

		// Track highest auto-name counter.
		m.trackCounter(p.Name)

		log.Info().Str("session", id.String()).Msg("restored session")
	}

	return nil
}

// CheckIdleSessions destroys sessions that have exceeded their TTL.
func (m *Manager) CheckIdleSessions() {
	m.mu.Lock()
	now := time.Now()
	var toDestroy []SessionID
	for id, sess := range m.sessions {
		if sess.TTL > 0 && now.Sub(sess.LastActive) > sess.TTL {
			toDestroy = append(toDestroy, id)
		}
	}
	m.mu.Unlock()

	for _, id := range toDestroy {
		log.Info().Str("session", id.String()).Msg("destroying idle session (TTL exceeded)")
		if err := m.DestroySession(id); err != nil {
			log.Warn().Err(err).Str("session", id.String()).Msg("failed to destroy idle session")
		}
	}
}

// Stop destroys all sessions and cleans up.
func (m *Manager) Stop() {
	m.mu.Lock()
	ids := make([]SessionID, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		_ = m.DestroySession(id)
	}
}

func (m *Manager) generateName(project string) string {
	for {
		m.counter++
		name := fmt.Sprintf("%s%d", autoNamePrefix, m.counter)
		id := SessionID{Project: project, Name: name}
		if _, exists := m.sessions[id]; !exists {
			return name
		}
	}
}

func (m *Manager) validateName(name string, proj config.Project) error {
	for _, svc := range proj.Services {
		if svc.Subdomain == name {
			return fmt.Errorf("%w: %q matches service subdomain", ErrNameConflict, name)
		}
	}
	return nil
}

func (m *Manager) trackCounter(name string) {
	var n int
	if _, err := fmt.Sscanf(name, autoNamePrefix+"%d", &n); err == nil {
		if n > m.counter {
			m.counter = n
		}
	}
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// StaticPorts extracts ports from all proxy URLs in the config.
func StaticPorts(cfg config.Config) []int {
	var ports []int
	for _, proj := range cfg.Projects {
		for _, svc := range proj.Services {
			if svc.Proxy == "" {
				continue
			}
			u, err := url.Parse(svc.Proxy)
			if err != nil {
				continue
			}
			port := u.Port()
			if port == "" {
				continue
			}
			if p, err := strconv.Atoi(port); err == nil {
				ports = append(ports, p)
			}
		}
	}
	return ports
}
