package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/daemon"
)

var downCmd = &cobra.Command{
	Use:     "down",
	Aliases: []string{"stop"},
	Short:   "Stop the Hatch daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDown()
	},
}

func runDown() error {
	// Check if the launchd job is loaded (idempotent).
	if !daemon.IsLoaded() {
		fmt.Printf("%s is not running\n", color.CyanString("Hatch"))
		return nil
	}

	// Stop the tray app first so it doesn't try to restart the daemon.
	stopTray()

	// Remove plist first so KeepAlive cannot respawn the process.
	if err := daemon.UninstallPlist(); err != nil {
		return fmt.Errorf("uninstall plist: %w", err)
	}
	log.Debug().Msg("plist removed")

	// Remove the job by label. This works after the plist file is deleted,
	// unlike "launchctl unload" which requires the file to exist.
	if err := daemon.RemoveJob(); err != nil {
		return fmt.Errorf("remove job: %w", err)
	}
	log.Debug().Msg("job removed")

	// Wait for the process to fully exit. launchctl remove sends SIGTERM
	// but returns immediately; the process may still be shutting down.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		running, _, _ := daemon.IsRunning()
		if !running {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("%s stopped\n", color.New(color.FgCyan, color.Bold).Sprint("Hatch"))
	return nil
}

// stopTray verifies the tray is running via its flock, reads the PID, and
// sends SIGTERM. The flock check prevents signaling a recycled PID.
func stopTray() {
	lockPath := filepath.Join(config.Dir(), "tray.lock")
	f, err := os.Open(lockPath)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	// Try to acquire the lock. If we get it, no tray is running.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		return
	}

	// Lock is held by the tray process. Read PID from the locked fd.
	data, err := io.ReadAll(io.LimitReader(f, 32))
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		log.Debug().Err(err).Int("pid", pid).Msg("could not signal tray process")
		return
	}
	log.Debug().Int("pid", pid).Msg("sent SIGTERM to tray")
}

func init() {
	rootCmd.AddCommand(downCmd)
}
