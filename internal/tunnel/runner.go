package tunnel

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"syscall"
	"time"
)

const (
	stopTimeout  = 5 * time.Second
	readyTimeout = 30 * time.Second
)

var trycloudflareRe = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

// RunnerConfig holds configuration for a tunnel runner.
type RunnerConfig struct {
	// Upstream is the local address to tunnel to (e.g. "http://localhost:3000").
	Upstream string
	// TunnelName is the named tunnel ID. Empty for quick tunnels.
	TunnelName string
	// Token is the Cloudflare API token for named tunnels.
	Token string
	// CloudflaredPath is the path to the cloudflared binary.
	CloudflaredPath string
	// OnOutput receives output lines from cloudflared.
	OnOutput func(source, line string)
}

// IsQuick returns true if this is a quick tunnel (no tunnel name).
func (cfg RunnerConfig) IsQuick() bool {
	return cfg.TunnelName == ""
}

// Runner manages the lifecycle of a single cloudflared tunnel process.
type Runner struct {
	cfg   RunnerConfig
	cmd   *exec.Cmd
	proxy *rewriteProxy
	done  chan struct{}
	err   error
	url   string
	mu    sync.Mutex
}

// Start launches the cloudflared process.
// For quick tunnels, it blocks until the tunnel URL is discovered or times out.
func (r *Runner) Start() error {
	r.mu.Lock()

	// For quick tunnels, start a rewrite proxy that strips absolute localhost
	// URLs from HTML responses. This prevents PNA (Private Network Access)
	// errors when frameworks like Vite inject absolute localhost script tags.
	if r.cfg.IsQuick() {
		proxy, err := startRewriteProxy(r.cfg.Upstream)
		if err != nil {
			r.mu.Unlock()
			return fmt.Errorf("starting rewrite proxy: %w", err)
		}
		r.proxy = proxy
	}

	cfUpstream := r.cfg.Upstream
	if r.proxy != nil {
		cfUpstream = r.proxy.addr
	}

	var cmd *exec.Cmd
	if r.cfg.IsQuick() {
		cmd = exec.Command(r.cfg.CloudflaredPath, "tunnel", "--url", cfUpstream)
	} else {
		cmd = exec.Command(r.cfg.CloudflaredPath, "tunnel", "run", r.cfg.TunnelName)
		// Pass token via environment to avoid exposing it in process args.
		cmd.Env = append(os.Environ(), "TUNNEL_TOKEN="+r.cfg.Token)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	closeProxy := func() {
		if r.proxy != nil {
			r.proxy.Close()
			r.proxy = nil
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		closeProxy()
		r.mu.Unlock()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		closeProxy()
		r.mu.Unlock()
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		closeProxy()
		r.mu.Unlock()
		return fmt.Errorf("start cloudflared: %w", err)
	}

	r.cmd = cmd
	r.done = make(chan struct{})
	done := r.done

	urlFound := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(2)

	scanOutput := func(scanner *bufio.Scanner, source string) {
		defer wg.Done()
		for scanner.Scan() {
			line := scanner.Text()
			if r.cfg.OnOutput != nil {
				r.cfg.OnOutput(source, line)
			}
			if r.cfg.IsQuick() {
				if match := trycloudflareRe.FindString(line); match != "" {
					select {
					case urlFound <- match:
					default:
					}
				}
			}
		}
	}

	go scanOutput(bufio.NewScanner(stdout), "stdout")
	go scanOutput(bufio.NewScanner(stderr), "stderr")

	go func() {
		wg.Wait()
		r.mu.Lock()
		r.err = cmd.Wait()
		r.mu.Unlock()
		close(done)
	}()

	// Release the lock before waiting on channels.
	r.mu.Unlock()

	if r.cfg.IsQuick() {
		select {
		case url := <-urlFound:
			r.mu.Lock()
			r.url = url
			r.mu.Unlock()
		case <-done:
			r.mu.Lock()
			closeProxy()
			r.mu.Unlock()
			return fmt.Errorf("cloudflared exited before providing tunnel URL")
		case <-time.After(readyTimeout):
			_ = r.Stop()
			return fmt.Errorf("timed out waiting for tunnel URL after %s", readyTimeout)
		}
	}

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

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return nil
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)

	select {
	case <-done:
	case <-time.After(stopTimeout):
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		<-done
	}

	r.mu.Lock()
	if r.proxy != nil {
		r.proxy.Close()
		r.proxy = nil
	}
	r.mu.Unlock()

	return nil
}

// Done returns a channel that is closed when the process exits.
func (r *Runner) Done() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.done
}

// URL returns the tunnel URL.
func (r *Runner) URL() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.url
}
