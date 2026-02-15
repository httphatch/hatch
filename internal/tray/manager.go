//go:build darwin

package tray

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"

	"github.com/pkg/browser"
	"github.com/rs/zerolog/log"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/daemon"
	"github.com/httphatch/hatch/internal/health"
)

// ManagerConfig holds the dependencies for a Manager.
type ManagerConfig struct {
	Version string
	App     *application.App
	Window  *application.WebviewWindow
	Icon    []byte
}

// Manager orchestrates the system tray icon, menu, and health polling.
type Manager struct {
	version string
	app     *application.App
	window  *application.WebviewWindow
	icon    []byte
	tray    *application.SystemTray
	checker *health.Checker

	mu   sync.Mutex
	done chan struct{}
}

// NewManager creates a Manager but does not start it.
func NewManager(cfg ManagerConfig) *Manager {
	return &Manager{
		version: cfg.Version,
		app:     cfg.App,
		window:  cfg.Window,
		icon:    cfg.Icon,
	}
}

// Start initialises the tray icon and begins periodic health polling.
func (m *Manager) Start() {
	m.tray = m.app.SystemTray.New()
	m.tray.SetTemplateIcon(iconPNG)

	cfg, err := config.Load()
	if err != nil {
		log.Warn().Err(err).Msg("tray: failed to load config")
		cfg = config.DefaultConfig()
	}

	m.checker = health.NewChecker(health.CheckerConfig{
		Interval: 5 * time.Second,
		Timeout:  2 * time.Second,
		OnChange: func(_ health.ServiceKey, _, _ health.Status) {
			m.refresh()
		},
	})
	if err := m.checker.Start(cfg); err != nil {
		log.Warn().Err(err).Msg("tray: failed to start health checker")
	}

	// Hide window on close instead of destroying it — removes the
	// Dock icon but keeps the tray icon running.
	if m.window != nil {
		m.window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
			m.hideWindow()
			e.Cancel()
		})
	}

	m.done = make(chan struct{})

	// Initial refresh.
	m.refresh()

	// Periodic refresh goroutine.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-m.done:
				return
			case <-ticker.C:
				m.refresh()
			}
		}
	}()
}

// Stop tears down the health checker and removes the tray icon.
func (m *Manager) Stop() {
	if m.done != nil {
		close(m.done)
	}
	if m.checker != nil {
		m.checker.Stop()
	}
	if m.tray != nil {
		m.tray.Destroy()
	}
}

// refresh reloads config, queries health, and rebuilds the menu.
func (m *Manager) refresh() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, err := config.Load()
	if err != nil {
		log.Debug().Err(err).Msg("tray: config load failed during refresh")
		cfg = config.DefaultConfig()
	}

	// Update the checker's targets in case config changed.
	if m.checker != nil {
		m.checker.UpdateConfig(cfg)
	}

	running, pid, _ := daemon.IsRunning()
	statuses := make(map[health.ServiceKey]health.ServiceStatus)
	if m.checker != nil {
		statuses = m.checker.ServiceStatuses()
	}

	// Build menu.
	menu := m.app.NewMenu()

	// Version header.
	menu.Add(fmt.Sprintf("Hatch %s", m.version)).SetEnabled(false)

	// Daemon status.
	if running {
		menu.Add(fmt.Sprintf("Daemon: Running (pid %d)", pid)).SetEnabled(false)
	} else {
		menu.Add("Daemon: Stopped").SetEnabled(false)
	}

	menu.AddSeparator()

	// Projects — sorted by name.
	projectNames := make([]string, 0, len(cfg.Projects))
	for name := range cfg.Projects {
		projectNames = append(projectNames, name)
	}
	sort.Strings(projectNames)

	for _, name := range projectNames {
		proj := cfg.Projects[name]
		m.buildProjectItem(menu, name, proj, statuses)
	}

	if len(projectNames) > 0 {
		menu.AddSeparator()
	}

	// Open Dashboard.
	menu.Add("Open Dashboard").OnClick(func(_ *application.Context) {
		m.showWindow()
	})

	// Add Project.
	menu.Add("Add Project...").OnClick(func(_ *application.Context) {
		m.showWindow()
	})

	menu.AddSeparator()

	// Daemon control.
	if running {
		menu.Add("Stop Daemon").OnClick(func(_ *application.Context) {
			go m.stopDaemon()
		})
	} else {
		menu.Add("Start Daemon").OnClick(func(_ *application.Context) {
			go m.startDaemon()
		})
	}

	menu.Add("Restart Daemon").OnClick(func(_ *application.Context) {
		go m.restartDaemon()
	})

	menu.AddSeparator()

	// Quit.
	menu.Add("Quit").OnClick(func(_ *application.Context) {
		m.app.Quit()
	})

	m.tray.SetMenu(menu)
}

// buildProjectItem adds a submenu item for a single project.
func (m *Manager) buildProjectItem(menu *application.Menu, name string, proj config.Project, statuses map[health.ServiceKey]health.ServiceStatus) {
	var dot string
	if !proj.Enabled {
		dot = "○"
	} else {
		allHealthy := true
		for svcName := range proj.Services {
			key := health.ServiceKey{Project: name, Service: svcName}
			if s, ok := statuses[key]; ok && s.Status != health.StatusHealthy {
				allHealthy = false
				break
			}
		}
		if allHealthy {
			dot = "●"
		} else {
			dot = "◐"
		}
	}

	sub := menu.AddSubmenu(fmt.Sprintf("%s %s", dot, proj.Domain))

	// Copy Domain.
	domain := proj.Domain
	sub.Add("Copy Domain").OnClick(func(_ *application.Context) {
		copyToClipboard(domain)
	})

	// Open in Browser.
	sub.Add("Open in Browser").OnClick(func(_ *application.Context) {
		u := url.URL{Scheme: "https", Host: domain}
		browser.OpenURL(u.String())
	})

	// Enable / Disable toggle.
	projName := name
	if proj.Enabled {
		sub.Add("Disable").OnClick(func(_ *application.Context) {
			go m.toggleProject(projName, false)
		})
	} else {
		sub.Add("Enable").OnClick(func(_ *application.Context) {
			go m.toggleProject(projName, true)
		})
	}

	sub.AddSeparator()

	// Per-service health rows.
	svcNames := make([]string, 0, len(proj.Services))
	for sn := range proj.Services {
		svcNames = append(svcNames, sn)
	}
	sort.Strings(svcNames)

	for _, svcName := range svcNames {
		svc := proj.Services[svcName]
		key := health.ServiceKey{Project: name, Service: svcName}
		indicator := "…"
		if s, ok := statuses[key]; ok {
			switch s.Status {
			case health.StatusHealthy:
				indicator = "✓"
			case health.StatusUnhealthy:
				indicator = "✗"
			}
		}
		addr := svc.Proxy
		sub.Add(fmt.Sprintf("%s  %s  %s", svcName, addr, indicator)).SetEnabled(false)
	}
}

func (m *Manager) showWindow() {
	if m.window != nil {
		setDockVisible(true)
		setAppIcon(m.icon)
		m.window.Show()
		m.window.SetAlwaysOnTop(true)
		m.window.SetAlwaysOnTop(false)
	}
}

func (m *Manager) hideWindow() {
	if m.window != nil {
		m.window.Hide()
		setDockVisible(false)
	}
}

// ── Actions ─────────────────────────────────────────────────────────────────

func (m *Manager) toggleProject(name string, enabled bool) {
	cfg, err := config.Load()
	if err != nil {
		log.Warn().Err(err).Msg("tray: config load failed")
		return
	}
	proj, ok := cfg.Projects[name]
	if !ok {
		return
	}
	proj.Enabled = enabled
	cfg.Projects[name] = proj
	if err := config.Save(cfg); err != nil {
		log.Warn().Err(err).Msg("tray: config save failed")
		return
	}
	m.restartDaemon()
}

func (m *Manager) stopDaemon() {
	m.runHatch("down")
	time.Sleep(500 * time.Millisecond)
	m.refresh()
}

func (m *Manager) startDaemon() {
	m.runHatch("up")
	time.Sleep(500 * time.Millisecond)
	m.refresh()
}

func (m *Manager) restartDaemon() {
	m.runHatch("restart")
	time.Sleep(500 * time.Millisecond)
	m.refresh()
}

func (m *Manager) runHatch(args ...string) {
	exe, err := os.Executable()
	if err != nil {
		log.Warn().Err(err).Msg("tray: failed to find executable path")
		return
	}
	cmd := exec.Command(exe, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Warn().Err(err).Str("output", string(out)).Msgf("tray: hatch %s failed", args[0])
	}
}

// copyToClipboard writes text to the macOS pasteboard via pbcopy.
func copyToClipboard(text string) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewReader([]byte(text))
	if err := cmd.Run(); err != nil {
		log.Warn().Err(err).Msg("tray: clipboard copy failed")
	}
}
