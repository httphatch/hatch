package health

import (
	"context"
	"errors"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/httphatch/hatch/internal/config"
)

// CheckerConfig controls the behaviour of a Checker.
type CheckerConfig struct {
	Interval time.Duration
	Timeout  time.Duration
	OnChange func(key ServiceKey, from, to Status) // optional transition callback
}

// Checker periodically TCP-dials upstream services and tracks their health.
type Checker struct {
	cfg      CheckerConfig
	mu       sync.Mutex
	targets  map[ServiceKey]string         // key → dial address
	statuses map[ServiceKey]*ServiceStatus // key → current status
	done     chan struct{}
	wg       sync.WaitGroup
	running  bool
}

// NewChecker creates a Checker with the given configuration.
// Zero-value Interval and Timeout are replaced with defaults.
func NewChecker(cfg CheckerConfig) *Checker {
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultInterval
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	return &Checker{
		cfg:      cfg,
		targets:  make(map[ServiceKey]string),
		statuses: make(map[ServiceKey]*ServiceStatus),
	}
}

// Start extracts dial targets from the application config, runs an immediate
// health check, and launches a background goroutine that re-checks on each
// tick of Interval. Returns an error if the checker is already running.
func (c *Checker) Start(appCfg config.Config) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return errors.New("checker already running")
	}
	c.done = make(chan struct{})
	c.running = true
	c.applyConfig(appCfg) // populates targets + statuses (called under lock)
	c.mu.Unlock()

	c.wg.Add(1)
	go c.loop()

	return nil
}

// Stop signals the background goroutine to exit and waits for it.
// It is a no-op if the checker is not running.
func (c *Checker) Stop() error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	close(c.done)
	c.running = false
	c.mu.Unlock()

	c.wg.Wait()
	return nil
}

// UpdateConfig adds, removes, or updates dial targets to match the new
// application config. Existing statuses are preserved for unchanged services.
func (c *Checker) UpdateConfig(appCfg config.Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.applyConfig(appCfg)
}

// ServiceStatuses returns a snapshot of all service statuses.
// The returned map contains value copies, safe to read without holding a lock.
func (c *Checker) ServiceStatuses() map[ServiceKey]ServiceStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make(map[ServiceKey]ServiceStatus, len(c.statuses))
	for k, v := range c.statuses {
		out[k] = *v
	}
	return out
}

// ServiceStatus returns the current status for a single service.
func (c *Checker) ServiceStatus(key ServiceKey) (ServiceStatus, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, ok := c.statuses[key]
	if !ok {
		return ServiceStatus{}, false
	}
	return *s, true
}

// loop runs an immediate check then re-checks on every ticker tick.
func (c *Checker) loop() {
	defer c.wg.Done()

	c.checkAll()

	ticker := time.NewTicker(c.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.checkAll()
		}
	}
}

// checkAll snapshots the current targets under lock, then dials each one
// without holding the lock to avoid blocking readers during slow dials.
func (c *Checker) checkAll() {
	c.mu.Lock()
	snap := make(map[ServiceKey]string, len(c.targets))
	for k, v := range c.targets {
		snap[k] = v
	}
	c.mu.Unlock()

	// Cap the total time for a full check cycle.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for key, addr := range snap {
		if ctx.Err() != nil {
			log.Warn().Msg("health check cycle timed out, skipping remaining targets")
			return
		}
		c.checkOne(ctx, key, addr)
	}
}

// checkOne dials a single target, updates its status, and fires the
// OnChange callback when the status transitions.
func (c *Checker) checkOne(ctx context.Context, key ServiceKey, addr string) {
	dialer := net.Dialer{Timeout: c.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if conn != nil {
		_ = conn.Close()
	}

	now := time.Now()
	newStatus := StatusHealthy
	if err != nil {
		newStatus = StatusUnhealthy
	}

	c.mu.Lock()
	ss, ok := c.statuses[key]
	if !ok {
		c.mu.Unlock()
		return // service removed while checking
	}

	oldStatus := ss.Status
	ss.LastCheck = now
	if oldStatus != newStatus {
		ss.Status = newStatus
		ss.Since = now
	}
	c.mu.Unlock()

	if oldStatus != newStatus {
		log.Info().
			Str("project", key.Project).
			Str("service", key.Service).
			Str("addr", addr).
			Str("from", oldStatus.String()).
			Str("to", newStatus.String()).
			Msg("service health changed")

		if c.cfg.OnChange != nil {
			c.cfg.OnChange(key, oldStatus, newStatus)
		}
	}
}

// applyConfig builds the targets map from the application config and
// initialises statuses for new services. Must be called with c.mu held.
func (c *Checker) applyConfig(appCfg config.Config) {
	newTargets := make(map[ServiceKey]string)

	for projName, proj := range appCfg.Projects {
		if !proj.Enabled {
			continue
		}
		for svcName, svc := range proj.Services {
			key := ServiceKey{Project: projName, Service: svcName}
			newTargets[key] = extractDialAddress(svc.Proxy)
		}
	}

	// Remove services that no longer exist.
	for key := range c.targets {
		if _, exists := newTargets[key]; !exists {
			delete(c.statuses, key)
		}
	}

	// Add or update services.
	now := time.Now()
	for key, addr := range newTargets {
		if existing, ok := c.statuses[key]; ok {
			// Update address if it changed.
			existing.Addr = addr
		} else {
			c.statuses[key] = &ServiceStatus{
				Status: StatusUnknown,
				Addr:   addr,
				Since:  now,
			}
		}
	}

	c.targets = newTargets
}

// extractDialAddress parses a proxy URL and returns the host:port dial
// address. For URLs without an explicit port, it defaults to :80 for http
// and :443 for https. Duplicated from caddy/translate.go to avoid a
// cross-package dependency.
func extractDialAddress(proxyURL string) string {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return proxyURL
	}
	host := u.Host
	if u.Port() == "" {
		switch u.Scheme {
		case "https":
			host += ":443"
		default:
			host += ":80"
		}
	}
	return host
}
