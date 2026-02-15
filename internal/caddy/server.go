package caddy

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	caddyv2 "github.com/caddyserver/caddy/v2"
	_ "github.com/caddyserver/caddy/v2/modules/standard"
)

// Server manages the lifecycle of an embedded Caddy instance.
type Server struct {
	cfg     ServerConfig
	client  *Client
	mu      sync.Mutex
	running bool
}

// NewServer creates a Server that will run Caddy with the given config.
// It wires up an internal Client sharing the same AdminAddr.
func NewServer(cfg ServerConfig) *Server {
	return &Server{
		cfg: cfg,
		client: &Client{
			AdminAddr:  cfg.AdminAddr,
			HTTPClient: &http.Client{},
		},
	}
}

// Start launches Caddy with a minimal admin-only configuration and waits
// until the admin API is ready. It returns an error if Caddy is already
// running or fails to start.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("caddy server already running")
	}
	s.mu.Unlock()

	// Redirect Caddy's data directory to ~/.hatch/caddy/ before first use.
	if err := ConfigureDataDir(); err != nil {
		return fmt.Errorf("configuring caddy data dir: %w", err)
	}

	if err := caddyv2.Run(&caddyv2.Config{
		Admin: &caddyv2.AdminConfig{
			Listen: s.cfg.AdminAddr,
		},
	}); err != nil {
		return fmt.Errorf("starting caddy: %w", err)
	}

	if err := s.waitForAdmin(ctx); err != nil {
		_ = caddyv2.Stop()
		return fmt.Errorf("waiting for caddy admin API: %w", err)
	}

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	return nil
}

// Stop shuts down the running Caddy instance. It is a no-op if Caddy
// is not running.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if err := caddyv2.Stop(); err != nil {
		return fmt.Errorf("stopping caddy: %w", err)
	}

	s.running = false
	return nil
}

// LoadConfig pushes a full Caddy JSON configuration via the admin API,
// delegating to the internal Client.
func (s *Server) LoadConfig(ctx context.Context, caddyConfig map[string]any) error {
	return s.client.Load(ctx, caddyConfig)
}

// waitForAdmin polls the admin API until it responds or the context
// is cancelled.
func (s *Server) waitForAdmin(ctx context.Context) error {
	reqURL := fmt.Sprintf("http://%s/config/", s.cfg.AdminAddr)

	for i := 0; i < 50; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return err
		}

		resp, err := s.client.HTTPClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}

	return fmt.Errorf("caddy admin API not ready after polling")
}
