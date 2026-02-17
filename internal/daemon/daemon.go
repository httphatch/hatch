// Package daemon manages the Hatch background process lifecycle,
// including start, stop, and status operations.
package daemon

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/httphatch/hatch/internal/api"
	"github.com/httphatch/hatch/internal/caddy"
	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/dns"
	"github.com/httphatch/hatch/internal/health"
	"github.com/httphatch/hatch/internal/process"
)

// Daemon orchestrates all Hatch subsystems as a long-running background process.
type Daemon struct {
	caddy     *caddy.Server
	dns       *dns.Server
	health    *health.Checker
	processes *process.Manager
	watcher   *config.Watcher
	api       *api.Server
	pidFile   *os.File
	caPaths   certs.CAPaths
	cfg       config.Config
	mu        sync.Mutex
	running   bool
	version   string
	startTime time.Time
	logHub    *api.LogHub
}

// New creates a new Daemon instance with the given version and log hub.
func New(version string, logHub *api.LogHub) *Daemon {
	return &Daemon{
		version: version,
		logHub:  logHub,
	}
}

// Run starts all subsystems and blocks until ctx is cancelled.
// On context cancellation it performs a graceful shutdown.
func (d *Daemon) Run(ctx context.Context) error {
	return d.RunSubsystems(ctx, nil)
}

// RunSubsystems starts all daemon subsystems and blocks until ctx is cancelled.
// An optional DashboardShower is passed to the API server for the dashboard/show endpoint.
func (d *Daemon) RunSubsystems(ctx context.Context, dashboard api.DashboardShower) error {
	d.startTime = time.Now()

	// Write PID file (acquires flock).
	pidFile, err := WritePID()
	if err != nil {
		return fmt.Errorf("write pid: %w", err)
	}
	d.pidFile = pidFile
	log.Info().Int("pid", os.Getpid()).Msg("pid file written")

	// Load config.
	cfg, err := config.Load()
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

	// Load translated config into Caddy.
	caddyCfg := caddy.Translate(cfg, caddy.PKIPaths{
		RootCert:         d.caPaths.Cert,
		RootKey:          d.caPaths.Key,
		IntermediateCert: d.caPaths.IntermediateCert,
		IntermediateKey:  d.caPaths.IntermediateKey,
	}, caddy.DataDir())
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
		},
	})
	if err := procMgr.ApplyConfig(cfg); err != nil {
		d.shutdownPartial()
		return fmt.Errorf("apply process config: %w", err)
	}
	d.processes = procMgr
	log.Info().Msg("process manager started")

	// Start API server.
	apiSrv := api.NewServer(api.ServerConfig{
		Addr:      "127.0.0.1:42824",
		Health:    d.health,
		Processes: d.processes,
		Daemon:    d,
		Dashboard: dashboard,
		CAPaths:   d.caPaths,
		Version:   d.version,
		StartTime: d.startTime,
		LogHub:    d.logHub,
	})
	if err := apiSrv.Start(); err != nil {
		d.shutdownPartial()
		return fmt.Errorf("start api server: %w", err)
	}
	d.api = apiSrv
	log.Info().Str("addr", "127.0.0.1:42824").Msg("api server started")

	// Start config watcher.
	watcher, err := config.NewWatcher(d.onConfigReload)
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

// HealthChecker returns the daemon's health checker instance.
// It is safe to call from any goroutine; returns nil before subsystems start.
func (d *Daemon) HealthChecker() *health.Checker {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.health
}

// APIHandler returns the API server's HTTP handler for in-process use.
// Returns nil before the API server has started.
func (d *Daemon) APIHandler() http.Handler {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.api != nil {
		return d.api.Handler()
	}
	return nil
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

	caddyCfg := caddy.Translate(cfg, caddy.PKIPaths{
		RootCert:         d.caPaths.Cert,
		RootKey:          d.caPaths.Key,
		IntermediateCert: d.caPaths.IntermediateCert,
		IntermediateKey:  d.caPaths.IntermediateKey,
	}, caddy.DataDir())
	if err := d.caddy.LoadConfig(context.Background(), caddyCfg); err != nil {
		log.Error().Err(err).Msg("failed to reload caddy config")
		return
	}

	d.health.UpdateConfig(cfg)

	if d.processes != nil {
		if err := d.processes.ApplyConfig(cfg); err != nil {
			log.Error().Err(err).Msg("failed to apply process config")
		}
	}

	log.Info().Msg("config reloaded successfully")
}

// ReloadConfig loads the current config and applies it to Caddy and the health checker.
func (d *Daemon) ReloadConfig() error {
	cfg, err := config.Load()
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
