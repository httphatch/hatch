//go:build darwin

package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/httphatch/hatch/internal/api"
	"github.com/httphatch/hatch/internal/app"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/tray"
)

var trayShowDashboard bool

var trayCmd = &cobra.Command{
	Use:    "_tray",
	Short:  "Run the tray app (internal)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTray()
	},
}

func runTray() error {
	// Acquire an exclusive lock to prevent duplicate tray processes.
	lockFile, err := acquireTrayLock()
	if err != nil {
		return fmt.Errorf("another tray instance is already running")
	}
	defer func() {
		_ = lockFile.Close()
		_ = os.Remove(lockFile.Name())
	}()

	a := app.NewApp()

	frontendAssets, err := fs.Sub(embeddedAssets, "frontend/dist")
	if err != nil {
		return fmt.Errorf("loading frontend assets: %w", err)
	}

	client := tray.NewClient()

	// Reverse proxy for API requests to the daemon.
	daemonURL, err := url.Parse("http://" + api.DefaultAddr)
	if err != nil {
		return fmt.Errorf("parse daemon URL: %w", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(daemonURL)

	staticHandler := application.BundledAssetFileServer(frontendAssets)
	trayAPIHandler := newTrayAPIHandler(client)
	assetHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/tray/") {
			trayAPIHandler.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			proxy.ServeHTTP(w, r)
			return
		}
		staticHandler.ServeHTTP(w, r)
	})

	wailsApp := application.New(application.Options{
		Name: "Hatch",
		Icon: appIconData,
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

	mgr := tray.NewManager(tray.ManagerConfig{
		Version: version,
		App:     wailsApp,
		Window:  window,
		Icon:    appIconData,
		Client:  client,
	})

	mgr.Init()

	wailsApp.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(_ *application.ApplicationEvent) {
		mgr.Start()
		if trayShowDashboard {
			mgr.ShowDashboard()
		}
	})

	wailsApp.OnShutdown(func() {
		mgr.Stop()
	})

	if err := wailsApp.Run(); err != nil {
		return fmt.Errorf("tray app: %w", err)
	}
	return nil
}

// newTrayAPIHandler returns an http.Handler for /api/tray/ endpoints that
// manage daemon lifecycle (start/stop/restart) via the tray client.
func newTrayAPIHandler(client *tray.Client) http.Handler {
	mux := http.NewServeMux()

	handle := func(action string, fn func() error, status string) {
		mux.HandleFunc("/api/tray/"+action, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
				return
			}
			if err := fn(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
		})
	}

	handle("start", client.StartDaemon, "started")
	handle("stop", client.StopDaemon, "stopped")
	handle("restart", client.RestartDaemon, "restarted")

	return mux
}

// acquireTrayLock tries to obtain an exclusive file lock on the tray lock file.
// Returns the locked file on success, or an error if another tray is running.
func acquireTrayLock() (*os.File, error) {
	path := filepath.Join(config.Dir(), "tray.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

func init() {
	trayCmd.Flags().BoolVar(&trayShowDashboard, "show", false, "show dashboard on startup")
	rootCmd.AddCommand(trayCmd)
}
