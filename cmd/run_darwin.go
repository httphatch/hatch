//go:build darwin

package cmd

import (
	"context"

	"github.com/httphatch/hatch/internal/daemon"
)

func runDaemon(ctx context.Context, d *daemon.Daemon) error {
	return daemon.RunWithUI(ctx, d, embeddedAssets, appIconData)
}
