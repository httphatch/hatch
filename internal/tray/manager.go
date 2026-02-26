//go:build darwin

package tray

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/pkg/browser"
	"github.com/rs/zerolog/log"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/health"
)

// ManagerConfig holds the dependencies for a Manager.
type ManagerConfig struct {
	Version string
	App     *application.App
	Window  *application.WebviewWindow
	Icon    []byte
	Client  *Client
}

// Manager orchestrates the system tray icon, menu, and health polling.
type Manager struct {
	version string
	app     *application.App
	window  *application.WebviewWindow
	icon    []byte
	tray    *application.SystemTray
	menu    *application.Menu
	client  *Client

	mu   sync.Mutex
	wg   sync.WaitGroup
	done chan struct{}
}

// NewManager creates a Manager but does not start it.
func NewManager(cfg ManagerConfig) *Manager {
	return &Manager{
		version: cfg.Version,
		app:     cfg.App,
		window:  cfg.Window,
		icon:    cfg.Icon,
		client:  cfg.Client,
	}
}

// Init creates the tray icon and sets an initial menu. It must be called
// before wailsApp.Run() so that the tray's Run() is deferred until the
// event loop starts, at which point the menu's native NSMenu is properly
// initialised.
func (m *Manager) Init() {
	m.menu = m.app.NewMenu()
	m.tray = m.app.SystemTray.New()
	m.tray.SetTemplateIcon(iconPNG)

	// Build the initial menu content so the native NSMenu is populated
	// when the deferred Run() fires during app startup.
	m.populateMenu()
	m.tray.SetMenu(m.menu)
}

// Start begins periodic menu refresh. It must be called after the Wails
// event loop is running (e.g. in the ApplicationStarted handler).
func (m *Manager) Start() {
	// Hide window on close instead of destroying it — removes the
	// Dock icon but keeps the tray icon running.
	if m.window != nil {
		m.window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
			m.hideWindow()
			e.Cancel()
		})
	}

	m.done = make(chan struct{})
	done := m.done // capture for goroutine

	// Periodic refresh goroutine.
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				m.refresh()
			}
		}
	}()
}

// Stop tears down the tray icon. It is safe to call multiple times.
func (m *Manager) Stop() {
	m.mu.Lock()
	done := m.done
	m.done = nil
	m.mu.Unlock()

	if done != nil {
		close(done)
		m.wg.Wait()
	}
	if m.tray != nil {
		m.tray.Destroy()
	}
}

// ShowDashboard shows the dashboard window.
func (m *Manager) ShowDashboard() {
	m.showWindow()
}

// refresh reloads config, queries health, and rebuilds the menu.
func (m *Manager) refresh() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.populateMenu()
	m.menu.Update()
}

// populateMenu clears the existing menu and rebuilds its items from
// the current config and health state.
func (m *Manager) populateMenu() {
	cfg, err := config.LoadWithProjectConfigs()
	if err != nil {
		log.Debug().Err(err).Msg("tray: config load failed during refresh")
		cfg = config.DefaultConfig()
	}

	// Query daemon status over HTTP.
	daemonRunning, daemonVersion, _ := m.client.DaemonStatus()

	statuses := make(map[health.ServiceKey]health.ServiceStatus)
	if daemonRunning {
		if h, err := m.client.Health(); err == nil {
			statuses = h
		}
	}

	// Clear existing items and rebuild.
	m.menu.Clear()

	// Version header — use daemon version if available, fall back to tray version.
	displayVersion := m.version
	if daemonVersion != "" {
		displayVersion = daemonVersion
	}
	m.menu.Add(fmt.Sprintf("Hatch %s", displayVersion)).SetEnabled(false)

	// Daemon status.
	if daemonRunning {
		m.menu.Add("Daemon: Running").SetEnabled(false)
	} else {
		m.menu.Add("Daemon: Stopped").SetEnabled(false)
	}

	m.menu.AddSeparator()

	// Projects — sorted by name.
	projectNames := make([]string, 0, len(cfg.Projects))
	for name := range cfg.Projects {
		projectNames = append(projectNames, name)
	}
	sort.Strings(projectNames)

	for _, name := range projectNames {
		proj := cfg.Projects[name]
		m.buildProjectItem(m.menu, name, proj, statuses, daemonRunning)
	}

	if len(projectNames) > 0 {
		m.menu.AddSeparator()
	}

	// Open Dashboard.
	m.menu.Add("Open Dashboard").OnClick(func(_ *application.Context) {
		m.showWindow()
	})

	// Add Project.
	m.menu.Add("Add Project...").OnClick(func(_ *application.Context) {
		m.showWindow()
	})

	m.menu.AddSeparator()

	if daemonRunning {
		// Restart Hatch (unload + reload via launchd).
		m.menu.Add("Restart Hatch").OnClick(func(_ *application.Context) {
			go func() {
				if err := m.client.RestartDaemon(); err != nil {
					log.Warn().Err(err).Msg("tray: restart failed")
				}
			}()
		})

		// Stop Hatch (remove plist + unload via launchd).
		m.menu.Add("Stop Hatch").OnClick(func(_ *application.Context) {
			go func() {
				if err := m.client.StopDaemon(); err != nil {
					log.Warn().Err(err).Msg("tray: stop failed")
				}
				m.refresh()
			}()
		})
	} else {
		// Start Hatch (install plist + load via launchd).
		m.menu.Add("Start Hatch").OnClick(func(_ *application.Context) {
			go func() {
				if err := m.client.StartDaemon(); err != nil {
					log.Warn().Err(err).Msg("tray: start failed")
				}
				m.refresh()
			}()
		})
	}

	// Quit tray (leaves daemon running).
	m.menu.AddSeparator()
	m.menu.Add("Quit").OnClick(func(_ *application.Context) {
		m.app.Quit()
	})
}

// buildProjectItem adds a submenu item for a single project.
func (m *Manager) buildProjectItem(menu *application.Menu, name string, proj config.Project, statuses map[health.ServiceKey]health.ServiceStatus, daemonRunning bool) {
	var dot string
	switch {
	case !proj.Enabled, !daemonRunning:
		dot = "○"
	default:
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
		_ = browser.OpenURL(u.String())
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
	if err := m.client.ReloadConfig(); err != nil {
		log.Warn().Err(err).Msg("tray: reload config failed")
	}
	m.refresh()
}

// copyToClipboard writes text to the macOS pasteboard via pbcopy.
func copyToClipboard(text string) {
	safe := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewReader([]byte(safe))
	if err := cmd.Run(); err != nil {
		log.Warn().Err(err).Msg("tray: clipboard copy failed")
	}
}
