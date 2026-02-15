//go:build darwin

package cmd

import (
	"fmt"
	"io/fs"

	"github.com/httphatch/hatch/internal/app"
	"github.com/httphatch/hatch/internal/tray"
	"github.com/spf13/cobra"
	"github.com/wailsapp/wails/v3/pkg/application"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Launch the Hatch GUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		a := app.NewApp()

		frontendAssets, err := fs.Sub(embeddedAssets, "frontend/dist")
		if err != nil {
			return fmt.Errorf("loading frontend assets: %w", err)
		}

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
				TitleBar: application.MacTitleBarHiddenInsetUnified,
				InvisibleTitleBarHeight: 48,
			},
		})

		mgr := tray.NewManager(tray.ManagerConfig{
			Version: version,
			App:     wailsApp,
			Window:  window,
			Icon:    appIconData,
		})

		wailsApp.OnShutdown(func() {
			mgr.Stop()
		})

		mgr.Start()

		return wailsApp.Run()
	},
}

func init() {
	rootCmd.AddCommand(appCmd)
}
