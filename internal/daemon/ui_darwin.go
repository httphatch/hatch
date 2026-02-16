//go:build darwin

package daemon

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

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
			Handler: application.BundledAssetFileServer(frontendAssets),
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

	// Start the tray manager once the Wails event loop is ready.
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
