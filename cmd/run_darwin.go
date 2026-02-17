//go:build darwin

package cmd

import (
	"context"

	"github.com/httphatch/hatch/internal/daemon"
)

func runDaemon(ctx context.Context, d *daemon.Daemon, trayIcon bool) error {
	if trayIcon {
		return daemon.RunWithUI(ctx, d, embeddedAssets, appIconData)
	}
	return d.Run(ctx)
}
