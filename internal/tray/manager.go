//go:build darwin

package tray

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"
	"slices"
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

// projectMenuState holds references to menu items within a project submenu
// so they can be updated in place without rebuilding.
type projectMenuState struct {
	submenuItem  *application.MenuItem
	submenu      *application.Menu
	toggleItem   *application.MenuItem
	serviceItems map[string]*application.MenuItem
	prevServices []string
	prevDomain   string
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

	// Stored menu item references for in-place updates.
	versionItem  *application.MenuItem
	daemonItem   *application.MenuItem
	restartItem  *application.MenuItem
	stopItem     *application.MenuItem
	startItem    *application.MenuItem
	projectItems map[string]*projectMenuState

	// Previous state for change detection.
	prevProjectNames []string
	menuBuilt        bool

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

	cfg, daemonRunning, daemonVersion, statuses := m.fetchState()
	m.rebuildMenu(cfg, daemonRunning, daemonVersion, statuses)
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

// refresh reloads config, queries health, and updates the menu.
// It performs a full rebuild only when the project/service set changes;
// otherwise it updates labels and visibility in place to avoid destroying
// native NSMenuItems and their click handlers.
func (m *Manager) refresh() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop() nils m.done before destroying the tray. Skip menu operations
	// if shutdown is in progress to avoid racing with tray.Destroy().
	if m.done == nil {
		return
	}

	cfg, daemonRunning, daemonVersion, statuses := m.fetchState()
	if m.needsRebuild(cfg) {
		m.rebuildMenu(cfg, daemonRunning, daemonVersion, statuses)
		m.menu.Update() // Push structural changes to the native NSMenu.
	} else {
		m.updateLabels(cfg, daemonRunning, daemonVersion, statuses)
	}
}

// fetchState loads config, queries daemon status and health.
func (m *Manager) fetchState() (config.Config, bool, string, map[health.ServiceKey]health.ServiceStatus) {
	cfg, err := config.LoadWithProjectConfigs()
	if err != nil {
		log.Debug().Err(err).Msg("tray: config load failed during refresh")
		cfg = config.DefaultConfig()
	}

	daemonRunning, daemonVersion, _ := m.client.DaemonStatus()

	statuses := make(map[health.ServiceKey]health.ServiceStatus)
	if daemonRunning {
		if h, err := m.client.Health(); err == nil {
			statuses = h
		}
	}

	return cfg, daemonRunning, daemonVersion, statuses
}

// needsRebuild returns true if the menu structure has changed and a full
// rebuild is required. Structural changes include project or service
// additions/removals, and domain changes.
func (m *Manager) needsRebuild(cfg config.Config) bool {
	if !m.menuBuilt {
		return true
	}

	names := sortedProjectNames(cfg)
	if !slices.Equal(names, m.prevProjectNames) {
		return true
	}

	for _, name := range names {
		proj := cfg.Projects[name]
		ps, ok := m.projectItems[name]
		if !ok {
			return true
		}
		if proj.Domain != ps.prevDomain {
			return true
		}
		if !slices.Equal(sortedServiceNames(proj), ps.prevServices) {
			return true
		}
	}

	return false
}

// rebuildMenu destroys all menu items and recreates them from scratch.
// Called on first init and when the project/service set changes.
func (m *Manager) rebuildMenu(cfg config.Config, daemonRunning bool, daemonVersion string, statuses map[health.ServiceKey]health.ServiceStatus) {
	m.menu.Clear()

	idx := 0

	// Version header.
	m.versionItem = m.menu.Add(fmt.Sprintf("Hatch %s", m.displayVersion(daemonVersion)))
	m.versionItem.SetEnabled(false)
	idx++

	// Daemon status.
	m.daemonItem = m.menu.Add(daemonLabel(daemonRunning))
	m.daemonItem.SetEnabled(false)
	idx++

	m.menu.AddSeparator()
	idx++

	// Projects — sorted by name.
	projectNames := sortedProjectNames(cfg)
	m.projectItems = make(map[string]*projectMenuState, len(projectNames))

	for _, name := range projectNames {
		proj := cfg.Projects[name]
		dot := projectDot(name, proj, statuses, daemonRunning)
		sub := m.menu.AddSubmenu(fmt.Sprintf("%s %s", dot, proj.Domain))
		submenuItem := m.menu.ItemAt(idx)
		idx++

		ps := &projectMenuState{
			submenuItem:  submenuItem,
			submenu:      sub,
			serviceItems: make(map[string]*application.MenuItem),
			prevDomain:   proj.Domain,
		}

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

		// Enable / Disable toggle. The handler reads fresh config state
		// at click time, so it does not need to be rebound on each poll.
		projName := name
		ps.toggleItem = sub.Add(toggleLabel(proj.Enabled))
		ps.toggleItem.OnClick(func(_ *application.Context) {
			go m.toggleProject(projName)
		})

		sub.AddSeparator()

		// Per-service health rows.
		svcNames := sortedServiceNames(proj)
		for _, svcName := range svcNames {
			svc := proj.Services[svcName]
			key := health.ServiceKey{Project: name, Service: svcName}
			indicator := serviceIndicator(statuses, key)
			item := sub.Add(fmt.Sprintf("%s  %s  %s", svcName, svc.Proxy, indicator))
			item.SetEnabled(false)
			ps.serviceItems[svcName] = item
		}

		ps.prevServices = svcNames
		m.projectItems[name] = ps
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

	// Restart Hatch.
	m.restartItem = m.menu.Add("Restart Hatch")
	m.restartItem.OnClick(func(_ *application.Context) {
		go func() {
			if err := m.client.RestartDaemon(); err != nil {
				log.Warn().Err(err).Msg("tray: restart failed")
			}
		}()
	})

	// Stop Hatch.
	m.stopItem = m.menu.Add("Stop Hatch")
	m.stopItem.OnClick(func(_ *application.Context) {
		go func() {
			if err := m.client.StopDaemon(); err != nil {
				log.Warn().Err(err).Msg("tray: stop failed")
			}
			m.refresh()
		}()
	})

	// Start Hatch.
	m.startItem = m.menu.Add("Start Hatch")
	m.startItem.OnClick(func(_ *application.Context) {
		go func() {
			if err := m.client.StartDaemon(); err != nil {
				log.Warn().Err(err).Msg("tray: start failed")
			}
			m.refresh()
		}()
	})

	m.applyDaemonVisibility(daemonRunning)

	// Quit tray.
	m.menu.AddSeparator()
	m.menu.Add("Quit").OnClick(func(_ *application.Context) {
		m.app.Quit()
	})

	m.prevProjectNames = projectNames
	m.menuBuilt = true
}

// updateLabels updates menu item labels and visibility in place without
// destroying and recreating items. This is the hot path that runs every
// 5 seconds and preserves click handler references.
func (m *Manager) updateLabels(cfg config.Config, daemonRunning bool, daemonVersion string, statuses map[health.ServiceKey]health.ServiceStatus) {
	m.versionItem.SetLabel(fmt.Sprintf("Hatch %s", m.displayVersion(daemonVersion)))
	m.daemonItem.SetLabel(daemonLabel(daemonRunning))
	m.applyDaemonVisibility(daemonRunning)

	for name, ps := range m.projectItems {
		proj := cfg.Projects[name]
		dot := projectDot(name, proj, statuses, daemonRunning)
		ps.submenuItem.SetLabel(fmt.Sprintf("%s %s", dot, proj.Domain))

		ps.toggleItem.SetLabel(toggleLabel(proj.Enabled))

		// Update service health indicators.
		for svcName, item := range ps.serviceItems {
			svc := proj.Services[svcName]
			key := health.ServiceKey{Project: name, Service: svcName}
			indicator := serviceIndicator(statuses, key)
			item.SetLabel(fmt.Sprintf("%s  %s  %s", svcName, svc.Proxy, indicator))
		}
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func (m *Manager) displayVersion(daemonVersion string) string {
	if daemonVersion != "" {
		if len(daemonVersion) > 64 {
			daemonVersion = daemonVersion[:64]
		}
		return daemonVersion
	}
	return m.version
}

func daemonLabel(running bool) string {
	if running {
		return "Daemon: Running"
	}
	return "Daemon: Stopped"
}

func toggleLabel(enabled bool) string {
	if enabled {
		return "Disable"
	}
	return "Enable"
}

func projectDot(name string, proj config.Project, statuses map[health.ServiceKey]health.ServiceStatus, daemonRunning bool) string {
	if !proj.Enabled || !daemonRunning {
		return "○"
	}
	for svcName := range proj.Services {
		key := health.ServiceKey{Project: name, Service: svcName}
		if s, ok := statuses[key]; ok && s.Status != health.StatusHealthy {
			return "◐"
		}
	}
	return "●"
}

func serviceIndicator(statuses map[health.ServiceKey]health.ServiceStatus, key health.ServiceKey) string {
	if s, ok := statuses[key]; ok {
		switch s.Status {
		case health.StatusHealthy:
			return "✓"
		case health.StatusUnhealthy:
			return "✗"
		}
	}
	return "…"
}

func (m *Manager) applyDaemonVisibility(running bool) {
	m.restartItem.SetHidden(!running)
	m.stopItem.SetHidden(!running)
	m.startItem.SetHidden(running)
}

func sortedProjectNames(cfg config.Config) []string {
	names := make([]string, 0, len(cfg.Projects))
	for name := range cfg.Projects {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func sortedServiceNames(proj config.Project) []string {
	names := make([]string, 0, len(proj.Services))
	for name := range proj.Services {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// ── Window ──────────────────────────────────────────────────────────────────

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

func (m *Manager) toggleProject(name string) {
	cfg, err := config.Load()
	if err != nil {
		log.Warn().Err(err).Msg("tray: config load failed")
		return
	}
	proj, ok := cfg.Projects[name]
	if !ok {
		return
	}
	proj.Enabled = !proj.Enabled
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
