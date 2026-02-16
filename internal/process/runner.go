package process

import (
	"bufio"
	"context"
	"fmt"
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

// Start launches the process. The context is used only for cancellation
// signalling — the process is stopped via Stop(), not context cancellation.
func (r *Runner) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := exec.CommandContext(ctx, "sh", "-c", r.cfg.Command)
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
