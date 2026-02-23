//go:build darwin

package daemon

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/rs/zerolog/log"
)

// StopDaemon removes the launchd plist (to prevent KeepAlive restart)
// then unloads the job, which sends SIGTERM to the current process.
func (d *Daemon) StopDaemon() {
	go func() {
		if err := UninstallPlist(); err != nil {
			log.Warn().Err(err).Msg("stop: uninstall plist failed")
			return
		}
		if err := UnloadPlist(); err != nil {
			log.Warn().Err(err).Msg("stop: unload plist failed")
		}
	}()
}

// RestartDaemon spawns a detached shell that unloads and reloads the
// launchd plist. The shell runs in its own process group so it survives
// the SIGTERM sent to this process during unload.
func (d *Daemon) RestartDaemon() {
	go func() {
		path, err := PlistPath()
		if err != nil {
			log.Warn().Err(err).Msg("restart: plist path")
			return
		}
		if _, err := os.Stat(path); err != nil {
			log.Warn().Err(err).Msg("restart: plist not found")
			return
		}
		// Pass the plist path via environment variable to avoid shell injection.
		// The shell is required because unload kills this process; load must
		// run in a detached process that survives the SIGTERM.
		cmd := exec.Command("sh", "-c",
			`launchctl unload "$PLIST_PATH" && launchctl load "$PLIST_PATH"`)
		cmd.Env = append(os.Environ(), "PLIST_PATH="+path)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			log.Warn().Err(err).Msg("restart: spawn failed")
		}
	}()
}
