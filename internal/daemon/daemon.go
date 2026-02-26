// Package daemon manages the Hatch background process lifecycle,
// including start, stop, and status operations.
package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/httphatch/hatch/internal/api"
	"github.com/httphatch/hatch/internal/caddy"
	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/cloudflare"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/dns"
	"github.com/httphatch/hatch/internal/health"
	"github.com/httphatch/hatch/internal/process"
	"github.com/httphatch/hatch/internal/tunnel"
)

// Daemon orchestrates all Hatch subsystems as a long-running background process.
type Daemon struct {
	caddy          *caddy.Server
	dns            *dns.Server
	health         *health.Checker
	processes      *process.Manager
	tunnels        *tunnel.Manager
	watcher        *config.Watcher
	api            *api.Server
	pidFile        *os.File
	caPaths        certs.CAPaths
	cfg            config.Config
	rewriteProxies map[string]*tunnel.RewriteProxy
	mu             sync.Mutex
	running        bool
	version        string
	startTime      time.Time
	logHub         *api.LogHub
	outputHub      *api.ProcessOutputHub
}

// New creates a new Daemon instance with the given version and log hub.
func New(version string, logHub *api.LogHub) *Daemon {
	return &Daemon{
		version:   version,
		logHub:    logHub,
		outputHub: api.NewProcessOutputHub(),
	}
}

// Run starts all subsystems and blocks until ctx is cancelled.
// On context cancellation it performs a graceful shutdown.
func (d *Daemon) Run(ctx context.Context) error {
	d.startTime = time.Now()

	// Write PID file (acquires flock).
	pidFile, err := WritePID()
	if err != nil {
		return fmt.Errorf("write pid: %w", err)
	}
	d.pidFile = pidFile
	log.Info().Int("pid", os.Getpid()).Msg("pid file written")

	// Load config.
	cfg, err := config.LoadWithProjectConfigs()
	if err != nil {
		_ = RemovePID(d.pidFile)
		return fmt.Errorf("load config: %w", err)
	}
	d.cfg = cfg

	// Pre-flight: check that required ports are free.
	for _, port := range []int{cfg.Settings.HTTPPort, cfg.Settings.HTTPSPort} {
		info, err := CheckPort(port)
		if err != nil {
			log.Warn().Err(err).Int("port", port).Msg("could not check port availability")
		} else if info != nil {
			_ = RemovePID(d.pidFile)
			return fmt.Errorf("port conflict: port :%d in use by %s", port, info)
		}
	}

	// Resolve CA paths and verify the files exist.
	d.caPaths = certs.NewCAPaths(config.CertsDir())
	if !certs.CAExists(d.caPaths) {
		_ = RemovePID(d.pidFile)
		return fmt.Errorf("CA files not found at %s — run 'hatch up' to generate", config.CertsDir())
	}
	if !certs.IntermediateCAExists(d.caPaths) {
		_ = RemovePID(d.pidFile)
		return fmt.Errorf("intermediate CA files not found at %s — run 'hatch up' to generate", config.CertsDir())
	}

	// Start DNS server.
	dnsSrv, err := dns.NewServer(dns.ServerConfig{
		TLD:      cfg.Settings.TLD,
		ListenIP: dns.DefaultListenIP,
		Port:     dns.DefaultPort,
	})
	if err != nil {
		_ = RemovePID(d.pidFile)
		return fmt.Errorf("create dns server: %w", err)
	}
	if err := dnsSrv.Start(); err != nil {
		_ = RemovePID(d.pidFile)
		return fmt.Errorf("start dns: %w", err)
	}
	d.dns = dnsSrv
	log.Info().Str("tld", cfg.Settings.TLD).Int("port", dns.DefaultPort).Msg("dns server started")

	// Clear Caddy's cached PKI so it uses our intermediate CA.
	if err := caddy.ClearPKICache(); err != nil {
		log.Warn().Err(err).Msg("failed to clear caddy PKI cache")
	}

	// Start Caddy server.
	caddySrv := caddy.NewServer(caddy.ServerConfig{
		AdminAddr: caddy.DefaultAdminAddr,
	})
	if err := caddySrv.Start(ctx); err != nil {
		d.shutdownPartial()
		return fmt.Errorf("start caddy: %w", err)
	}
	d.caddy = caddySrv
	log.Info().Msg("caddy server started")

	// Resolve tunnel domains from Cloudflare API before building Caddy config.
	tunnelDomains := tunnel.ResolveTunnelDomains(cfg, cloudflare.NewClient())
	if len(tunnelDomains) > 0 {
		total := 0
		for _, hs := range tunnelDomains {
			total += len(hs)
		}
		log.Info().Int("domains", total).Int("services", len(tunnelDomains)).Msg("resolved tunnel domains")
	}

	// Start rewrite proxies for services with tunnel domains so that
	// named tunnel traffic through Caddy gets localhost URLs rewritten.
	tunnelProxies := d.startTunnelProxies(cfg, tunnelDomains)

	// Load translated config into Caddy.
	caddyCfg := caddy.Translate(cfg, caddy.PKIPaths{
		RootCert:         d.caPaths.Cert,
		RootKey:          d.caPaths.Key,
		IntermediateCert: d.caPaths.IntermediateCert,
		IntermediateKey:  d.caPaths.IntermediateKey,
	}, caddy.DataDir(), tunnelDomains, tunnelProxies)
	if err := caddySrv.LoadConfig(ctx, caddyCfg); err != nil {
		d.shutdownPartial()
		return fmt.Errorf("load caddy config: %w", err)
	}
	log.Info().Msg("caddy config loaded")

	// Start health checker.
	checker := health.NewChecker(health.CheckerConfig{})
	if err := checker.Start(cfg); err != nil {
		d.shutdownPartial()
		return fmt.Errorf("start health checker: %w", err)
	}
	d.health = checker
	log.Info().Msg("health checker started")

	// Start process manager.
	procMgr := process.NewManager(process.ManagerConfig{
		OnOutput: func(project, service, source, line string) {
			log.Info().
				Str("project", project).
				Str("service", service).
				Str("source", source).
				Msg(line)
			d.outputHub.Write(project, service, source, line)
		},
	})
	if err := procMgr.ApplyConfig(cfg); err != nil {
		d.shutdownPartial()
		return fmt.Errorf("apply process config: %w", err)
	}
	d.processes = procMgr
	log.Info().Msg("process manager started")

	// Start tunnel manager.
	stateFile := filepath.Join(config.Dir(), "tunnels.json")
	tunnelMgr := tunnel.NewManager(stateFile, func(project, service, source, line string) {
		log.Info().
			Str("project", project).
			Str("service", service).
			Str("source", source).
			Msg(line)
		d.outputHub.Write(project, service, source, line)
	})
	if err := tunnelMgr.ApplyConfig(cfg); err != nil {
		log.Warn().Err(err).Msg("failed to apply tunnel config")
	}
	d.tunnels = tunnelMgr
	log.Info().Msg("tunnel manager started")

	// Start API server.
	apiSrv := api.NewServer(api.ServerConfig{
		Addr:      api.DefaultAddr,
		Health:    d.health,
		Processes: d.processes,
		Tunnels:   d.tunnels,
		Daemon:    d,
		CAPaths:   d.caPaths,
		Version:   d.version,
		StartTime: d.startTime,
		LogHub:    d.logHub,
		OutputHub: d.outputHub,
	})
	if err := apiSrv.Start(); err != nil {
		d.shutdownPartial()
		return fmt.Errorf("start api server: %w", err)
	}
	d.api = apiSrv
	log.Info().Str("addr", api.DefaultAddr).Msg("api server started")

	// Start config watcher.
	watcher, err := config.NewWatcher(d.cfg, d.onConfigReload)
	if err != nil {
		d.shutdownPartial()
		return fmt.Errorf("start config watcher: %w", err)
	}
	d.watcher = watcher
	log.Info().Msg("config watcher started")

	d.mu.Lock()
	d.running = true
	d.mu.Unlock()

	log.Info().Msg("daemon running")

	// Block until context is cancelled.
	<-ctx.Done()
	log.Info().Msg("shutdown signal received")

	return d.Shutdown()
}

// Shutdown stops all subsystems in reverse start order and removes the PID file.
func (d *Daemon) Shutdown() error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = false
	d.mu.Unlock()

	var errs []error

	// API server first — stop accepting requests.
	if d.api != nil {
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := d.api.Shutdown(shutCtx); err != nil {
			errs = append(errs, fmt.Errorf("stop api server: %w", err))
		}
		cancel()
		log.Info().Msg("api server stopped")
	}

	// Watcher — prevents reload during teardown.
	if d.watcher != nil {
		if err := d.watcher.Close(); err != nil {
			errs = append(errs, fmt.Errorf("stop watcher: %w", err))
		}
		log.Info().Msg("config watcher stopped")
	}

	// Rewrite proxies (before tunnel manager).
	d.stopTunnelProxies()

	// Tunnel manager (before process manager so tunnels disconnect first).
	if d.tunnels != nil {
		if err := d.tunnels.StopAll(); err != nil {
			errs = append(errs, fmt.Errorf("stop tunnel manager: %w", err))
		}
		log.Info().Msg("tunnel manager stopped")
	}

	// Process manager.
	if d.processes != nil {
		if err := d.processes.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop process manager: %w", err))
		}
		log.Info().Msg("process manager stopped")
	}

	// Health checker.
	if d.health != nil {
		if err := d.health.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop health checker: %w", err))
		}
		log.Info().Msg("health checker stopped")
	}

	// Caddy.
	if d.caddy != nil {
		if err := d.caddy.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop caddy: %w", err))
		}
		log.Info().Msg("caddy server stopped")
	}

	// DNS.
	if d.dns != nil {
		if err := d.dns.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop dns: %w", err))
		}
		log.Info().Msg("dns server stopped")
	}

	// Remove PID file.
	if d.pidFile != nil {
		if err := RemovePID(d.pidFile); err != nil {
			errs = append(errs, fmt.Errorf("remove pid: %w", err))
		}
		log.Info().Msg("pid file removed")
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	log.Info().Msg("daemon stopped")
	return nil
}

// onConfigReload is called by the config watcher when the config file changes.
// It re-translates the Caddy config and updates the health checker.
func (d *Daemon) onConfigReload(cfg config.Config) {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	d.cfg = cfg
	d.mu.Unlock()

	tunnelDomains := tunnel.ResolveTunnelDomains(cfg, cloudflare.NewClient())

	// Start new proxies before stopping old ones so Caddy always has a
	// valid upstream to route to. Old proxies are stopped after Caddy
	// switches to the new config.
	oldProxies := d.swapTunnelProxies(cfg, tunnelDomains)

	d.mu.Lock()
	tunnelProxies := make(map[string]string, len(d.rewriteProxies))
	for k, p := range d.rewriteProxies {
		tunnelProxies[k] = p.Addr()
	}
	d.mu.Unlock()

	caddyCfg := caddy.Translate(cfg, caddy.PKIPaths{
		RootCert:         d.caPaths.Cert,
		RootKey:          d.caPaths.Key,
		IntermediateCert: d.caPaths.IntermediateCert,
		IntermediateKey:  d.caPaths.IntermediateKey,
	}, caddy.DataDir(), tunnelDomains, tunnelProxies)
	if err := d.caddy.LoadConfig(context.Background(), caddyCfg); err != nil {
		log.Error().Err(err).Msg("failed to reload caddy config")
		return
	}

	// Stop old proxies now that Caddy has switched to the new ones.
	for key, proxy := range oldProxies {
		proxy.Close()
		log.Info().Str("service", key).Msg("old rewrite proxy stopped")
	}

	d.health.UpdateConfig(cfg)

	if d.processes != nil {
		if err := d.processes.ApplyConfig(cfg); err != nil {
			log.Error().Err(err).Msg("failed to apply process config")
		}
	}

	if d.tunnels != nil {
		if err := d.tunnels.ApplyConfig(cfg); err != nil {
			log.Error().Err(err).Msg("failed to apply tunnel config")
		}
	}

	log.Info().Msg("config reloaded successfully")
}

// ReloadConfig loads the current config and applies it to Caddy and the health checker.
func (d *Daemon) ReloadConfig() error {
	cfg, err := config.LoadWithProjectConfigs()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	d.onConfigReload(cfg)
	return nil
}

// shutdownPartial stops any subsystems that were started during a failed Run.
func (d *Daemon) shutdownPartial() {
	if d.api != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = d.api.Shutdown(ctx)
		cancel()
	}
	d.stopTunnelProxies()
	if d.tunnels != nil {
		_ = d.tunnels.StopAll()
	}
	if d.processes != nil {
		_ = d.processes.Stop()
	}
	if d.health != nil {
		_ = d.health.Stop()
	}
	if d.caddy != nil {
		_ = d.caddy.Stop()
	}
	if d.dns != nil {
		_ = d.dns.Stop()
	}
	if d.pidFile != nil {
		_ = RemovePID(d.pidFile)
	}
}

// startTunnelProxies starts rewrite proxies for services that have tunnel
// domains. Returns a map of "project/service" keys to proxy addresses.
func (d *Daemon) startTunnelProxies(cfg config.Config, tunnelDomains map[string][]string) map[string]string {
	if len(tunnelDomains) == 0 {
		return nil
	}

	newProxies := make(map[string]*tunnel.RewriteProxy, len(tunnelDomains))
	addrs := make(map[string]string, len(tunnelDomains))

	for key := range tunnelDomains {
		upstream := upstreamForKey(cfg, key)
		if upstream == "" {
			continue
		}
		proxy, err := tunnel.StartRewriteProxy(upstream)
		if err != nil {
			log.Warn().Err(err).Str("service", key).Msg("failed to start rewrite proxy")
			continue
		}
		newProxies[key] = proxy
		addrs[key] = proxy.Addr()
		log.Info().Str("service", key).Str("addr", proxy.Addr()).Msg("rewrite proxy started")
	}

	d.mu.Lock()
	d.rewriteProxies = newProxies
	d.mu.Unlock()

	return addrs
}

// swapTunnelProxies starts new rewrite proxies and returns the old ones for
// the caller to close after Caddy has switched to the new config.
func (d *Daemon) swapTunnelProxies(cfg config.Config, tunnelDomains map[string][]string) map[string]*tunnel.RewriteProxy {
	d.mu.Lock()
	old := d.rewriteProxies
	d.rewriteProxies = nil
	d.mu.Unlock()

	d.startTunnelProxies(cfg, tunnelDomains)
	return old
}

// stopTunnelProxies shuts down all running rewrite proxies.
func (d *Daemon) stopTunnelProxies() {
	d.mu.Lock()
	proxies := d.rewriteProxies
	d.rewriteProxies = nil
	d.mu.Unlock()

	for key, proxy := range proxies {
		proxy.Close()
		log.Info().Str("service", key).Msg("rewrite proxy stopped")
	}
}

// upstreamForKey returns the proxy URL for a "project/service" key.
func upstreamForKey(cfg config.Config, key string) string {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	proj, ok := cfg.Projects[parts[0]]
	if !ok {
		return ""
	}
	svc, ok := proj.Services[parts[1]]
	if !ok {
		return ""
	}
	return svc.Proxy
}
