package tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Common install paths for cloudflared on macOS. The daemon runs under
// launchd which has a minimal $PATH that does not include Homebrew's
// bin directory, so exec.LookPath alone is not enough.
var fallbackPaths = []string{
	"/opt/homebrew/bin/cloudflared",
	"/usr/local/bin/cloudflared",
}

// FindCloudflared returns the path to the cloudflared binary.
func FindCloudflared() (string, error) {
	path, err := exec.LookPath("cloudflared")
	if err == nil {
		return path, nil
	}

	if runtime.GOOS == "darwin" {
		for _, p := range fallbackPaths {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	return "", fmt.Errorf("cloudflared not found (install with: brew install cloudflared): %w", err)
}
