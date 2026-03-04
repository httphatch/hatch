package process

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const (
	stopTimeout = 5 * time.Second
)

// RunnerConfig holds the configuration for a single managed process.
type RunnerConfig struct {
	Name     string
	Command  string
	Dir      string
	Env      []string
	OnOutput func(name, source, line string)
}

// Runner manages the lifecycle of a single child process.
type Runner struct {
	cfg  RunnerConfig
	cmd  *exec.Cmd
	done chan struct{}
	err  error
	mu   sync.Mutex
}

// Start launches the process. The process is stopped exclusively via Stop(),
// which sends SIGTERM to the entire process group for clean shutdown.
func (r *Runner) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	cmd := exec.Command(shell, "-l", "-c", r.cfg.Command)
	cmd.Dir = r.cfg.Dir
	cmd.Env = r.cfg.Env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	r.cmd = cmd
	r.done = make(chan struct{})

	// Scan output in background goroutines.
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			if r.cfg.OnOutput != nil {
				r.cfg.OnOutput(r.cfg.Name, "stdout", s.Text())
			}
		}
	}()

	go func() {
		defer wg.Done()
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			if r.cfg.OnOutput != nil {
				r.cfg.OnOutput(r.cfg.Name, "stderr", s.Text())
			}
		}
	}()

	// Wait for process exit in background.
	go func() {
		wg.Wait()
		r.mu.Lock()
		r.err = cmd.Wait()
		r.mu.Unlock()
		close(r.done)
	}()

	return nil
}

// Stop sends SIGTERM to the process group, waits up to 5 seconds,
// then sends SIGKILL if still running.
func (r *Runner) Stop() error {
	r.mu.Lock()
	cmd := r.cmd
	done := r.done
	r.mu.Unlock()

	if cmd == nil || cmd.Process == nil || done == nil {
		return nil
	}

	// Send SIGTERM to the process group.
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Process already exited.
		return nil
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)

	// Wait for graceful exit or timeout.
	select {
	case <-done:
		return nil
	case <-time.After(stopTimeout):
	}

	// Force kill.
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	<-done
	return nil
}

// PID returns the process ID of the managed process, or 0 if not started.
func (r *Runner) PID() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process.Pid
	}
	return 0
}

// Done returns a channel that is closed when the process exits.
func (r *Runner) Done() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.done
}

// ExitErr returns the process exit error, or nil if it exited successfully.
// Only valid after Done() is closed.
func (r *Runner) ExitErr() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}
