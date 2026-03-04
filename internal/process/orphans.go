package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

const orphanStopTimeout = 5 * time.Second

// CleanOrphans reads the PID state file and kills any surviving processes
// from a previous daemon instance. It removes the state file after cleanup.
func CleanOrphans(pidFile string) error {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading orphan PID file: %w", err)
	}

	var pids map[string]int
	if err := json.Unmarshal(data, &pids); err != nil {
		// Corrupted file; remove it and move on.
		_ = os.Remove(pidFile)
		return nil
	}

	if len(pids) == 0 {
		_ = os.Remove(pidFile)
		return nil
	}

	for name, pid := range pids {
		if pid <= 0 {
			log.Warn().Str("service", name).Int("pid", pid).Msg("ignoring invalid PID in state file")
			continue
		}
		killOrphan(name, pid)
	}

	_ = os.Remove(pidFile)
	return nil
}

func killOrphan(name string, pid int) {
	// Check if the process is still running.
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return
	}

	log.Info().Str("service", name).Int("pid", pid).Msg("killing orphaned process")

	// Send SIGTERM to the individual process. We avoid killing the process
	// group here because the orphaned PID may have been reused by the OS,
	// and killing an unknown process group would have excessive blast radius.
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		log.Warn().Err(err).Str("service", name).Int("pid", pid).Msg("failed to send SIGTERM to orphan")
		return
	}

	// Wait for the process to exit.
	deadline := time.After(orphanStopTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			// Re-check that the process is still alive before escalating.
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				log.Info().Str("service", name).Int("pid", pid).Msg("orphaned process terminated")
				return
			}
			log.Warn().Str("service", name).Int("pid", pid).Msg("orphaned process did not exit, sending SIGKILL")
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
				log.Warn().Err(err).Str("service", name).Int("pid", pid).Msg("failed to send SIGKILL to orphan")
			}
			return
		case <-ticker.C:
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				log.Info().Str("service", name).Int("pid", pid).Msg("orphaned process terminated")
				return
			}
		}
	}
}
