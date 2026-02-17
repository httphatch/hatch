//go:build darwin

package daemon

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/httphatch/hatch/internal/app"
	"github.com/httphatch/hatch/internal/tray"
)

// RunWithUI starts the daemon subsystems and the Wails tray/window in a single process.
// The Wails app runs on the main thread (macOS requirement); daemon subsystems run in a goroutine.
func RunWithUI(parentCtx context.Context, d *Daemon, assets embed.FS, icon []byte) error {
	// Derive a cancellable context so we can stop daemon subsystems when Wails quits.
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	a := app.NewApp()

	frontendAssets, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		return fmt.Errorf("loading frontend assets: %w", err)
	}

	// Route /api/ requests directly to the daemon's API handler so the
	// webview doesn't need cross-scheme HTTP fetches (wails:// → http://).
	// The handler is resolved per-request because subsystems start async.
	staticHandler := application.BundledAssetFileServer(frontendAssets)
	assetHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			// Wait briefly for subsystems to finish starting.
			for range 50 {
				if h := d.APIHandler(); h != nil {
					h.ServeHTTP(w, r)
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"daemon starting up"}`))
			return
		}
		staticHandler.ServeHTTP(w, r)
	})

	wailsApp := application.New(application.Options{
		Name: "Hatch",
		Icon: icon,
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
		Services: []application.Service{
			application.NewService(a),
		},
		Assets: application.AssetOptions{
			Handler: assetHandler,
		},
	})

	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Hatch",
		Width:     1024,
		Height:    768,
		MinWidth:  800,
		MinHeight: 600,
		Hidden:    true,
		Mac: application.MacWindow{
			TitleBar:                application.MacTitleBarHiddenInsetUnified,
			InvisibleTitleBarHeight: 48,
		},
	})

	// Create tray manager. The daemon is passed as DaemonControl which also
	// provides HealthChecker(). The health checker may be nil initially (before
	// subsystems finish starting) — the tray's refresh handles this gracefully.
	mgr := tray.NewManager(tray.ManagerConfig{
		Version: d.version,
		App:     wailsApp,
		Window:  window,
		Icon:    icon,
		Daemon:  d,
	})

	// Channel to communicate subsystem startup result.
	subsystemErr := make(chan error, 1)

	// Start daemon subsystems in a background goroutine.
	go func() {
		if err := d.RunSubsystems(ctx, mgr); err != nil {
			log.Error().Err(err).Msg("daemon subsystems failed")
			subsystemErr <- err
		} else {
			subsystemErr <- nil
		}
		// Subsystems stopped — quit Wails app.
		wailsApp.Quit()
	}()

	// Init creates the tray and sets the initial menu while the app is not
	// yet running, so the deferred Run() will see the menu and initialise
	// the native NSMenu pointer. Start() is called later once the event
	// loop is up to begin periodic refresh.
	mgr.Init()

	wailsApp.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(_ *application.ApplicationEvent) {
		mgr.Start()
	})

	wailsApp.OnShutdown(func() {
		mgr.Stop()
		cancel() // Stop daemon subsystems when Wails quits.
	})

	// wailsApp.Run() blocks on the main thread.
	if err := wailsApp.Run(); err != nil {
		select {
		case subErr := <-subsystemErr:
			if subErr != nil {
				return subErr
			}
		default:
		}
		return fmt.Errorf("wails app: %w", err)
	}

	// Return subsystem error if any.
	select {
	case subErr := <-subsystemErr:
		return subErr
	default:
		return nil
	}
}
