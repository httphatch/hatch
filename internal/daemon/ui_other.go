//go:build !darwin

package daemon

import (
	"context"
	"embed"
)

// RunWithUI is a no-op on non-darwin platforms — it just runs the daemon directly.
func RunWithUI(ctx context.Context, d *Daemon, _ embed.FS, _ []byte) error {
	return d.Run(ctx)
}
