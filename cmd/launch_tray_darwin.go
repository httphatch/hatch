//go:build darwin

package cmd

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/rs/zerolog/log"
)

// launchTray starts the tray app as a detached process. If show is true, the
// dashboard window opens on startup.
func launchTray(show bool) {
	bin, err := os.Executable()
	if err != nil {
		log.Warn().Err(err).Msg("could not resolve executable path for tray")
		return
	}
	args := []string{"_tray"}
	if show {
		args = append(args, "--show")
	}
	cmd := exec.Command(bin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		log.Warn().Err(err).Msg("failed to launch tray")
		return
	}
	go func() { _ = cmd.Wait() }()
}
