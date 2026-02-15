package daemon

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/httphatch/hatch/internal/config"
)

const pidFileName = "hatch.pid"

// PIDFile returns the path to the PID file (~/.hatch/hatch.pid).
func PIDFile() string {
	return filepath.Join(config.Dir(), pidFileName)
}

// WritePID creates the PID file and acquires an exclusive flock on it.
// The caller must keep the returned *os.File open for the lock lifetime.
// Closing the file or process exit releases the lock automatically.
func WritePID() (*os.File, error) {
	path := PIDFile()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open pid file: %w", err)
	}

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock pid file (another instance running?): %w", err)
	}

	if err := f.Truncate(0); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("truncate pid file: %w", err)
	}

	_, err = fmt.Fprintf(f, "%d\n", os.Getpid())
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("write pid: %w", err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("sync pid file: %w", err)
	}

	return f, nil
}

// RemovePID closes the file (releasing the flock) and removes the PID file.
func RemovePID(f *os.File) error {
	path := f.Name()
	if err := f.Close(); err != nil {
		return fmt.Errorf("close pid file: %w", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove pid file: %w", err)
	}
	return nil
}

// ReadPID reads the PID from the PID file.
func ReadPID() (int, error) {
	data, err := os.ReadFile(PIDFile())
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse pid: %w", err)
	}
	return pid, nil
}

// readPIDFrom reads the PID from an already-open file.
func readPIDFrom(f *os.File) (int, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	data, err := io.ReadAll(io.LimitReader(f, 32))
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse pid: %w", err)
	}
	return pid, nil
}

// IsRunning checks whether the daemon is running by attempting to acquire
// an flock on the PID file. If the lock cannot be acquired (EWOULDBLOCK),
// the daemon is running.
func IsRunning() (bool, int, error) {
	path := PIDFile()

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, err
	}
	defer func() { _ = f.Close() }()

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		// Lock held by another process — daemon is running.
		// Read PID from the already-open fd to avoid TOCTOU race.
		pid, readErr := readPIDFrom(f)
		if readErr != nil {
			return true, 0, nil
		}
		return true, pid, nil
	}

	// We got the lock — no daemon running. Release it.
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return false, 0, nil
}
