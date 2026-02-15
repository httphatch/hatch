package caddy

import (
	"context"
	"fmt"
	"net"
	"testing"
)

// freePort returns an available TCP port on 127.0.0.1.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

func TestServer_StartStop(t *testing.T) {
	port := freePort(t)
	srv := NewServer(ServerConfig{
		AdminAddr: fmt.Sprintf("localhost:%d", port),
	})

	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Admin API should be reachable — waitForAdmin already confirmed this
	// during Start, but verify the running flag is set.
	srv.mu.Lock()
	running := srv.running
	srv.mu.Unlock()
	if !running {
		t.Fatal("expected server to be marked as running")
	}

	if err := srv.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	srv.mu.Lock()
	running = srv.running
	srv.mu.Unlock()
	if running {
		t.Fatal("expected server to be marked as not running after Stop")
	}
}

func TestServer_DoubleStartFails(t *testing.T) {
	port := freePort(t)
	srv := NewServer(ServerConfig{
		AdminAddr: fmt.Sprintf("localhost:%d", port),
	})

	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("first Start failed: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	if err := srv.Start(ctx); err == nil {
		t.Fatal("expected error on second Start, got nil")
	}
}

func TestServer_StopWithoutStart(t *testing.T) {
	port := freePort(t)
	srv := NewServer(ServerConfig{
		AdminAddr: fmt.Sprintf("localhost:%d", port),
	})

	if err := srv.Stop(); err != nil {
		t.Fatalf("Stop without Start should be no-op, got: %v", err)
	}
}

func TestServer_LoadConfig(t *testing.T) {
	adminPort := freePort(t)
	httpPort := freePort(t)

	srv := NewServer(ServerConfig{
		AdminAddr: fmt.Sprintf("localhost:%d", adminPort),
	})

	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	caddyCfg := map[string]any{
		"admin": map[string]any{
			"listen": fmt.Sprintf("localhost:%d", adminPort),
		},
		"apps": map[string]any{
			"http": map[string]any{
				"servers": map[string]any{
					"test": map[string]any{
						"listen": []string{fmt.Sprintf(":%d", httpPort)},
						"routes": []map[string]any{
							{
								"handle": []map[string]any{
									{
										"handler":     "static_response",
										"status_code": "200",
										"body":        "ok",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := srv.LoadConfig(ctx, caddyCfg); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
}
