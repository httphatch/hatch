package tunnel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/httphatch/hatch/internal/config"
)

func TestQuickTunnelURLParsing(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "mock-cloudflared")
	err := os.WriteFile(script, []byte(`#!/bin/sh
echo "some startup output"
echo "https://test-abc-123.trycloudflare.com"
sleep 60
`), 0755)
	if err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	stateFile := filepath.Join(dir, "tunnels.json")
	mgr := NewManager(stateFile, nil)

	id := TunnelID{Project: "myapp", Service: "web"}

	runner := &Runner{cfg: RunnerConfig{
		Upstream:        "http://localhost:3000",
		CloudflaredPath: script,
	}}

	if err := runner.Start(); err != nil {
		t.Fatalf("starting runner: %v", err)
	}

	url := runner.URL()
	if url != "https://test-abc-123.trycloudflare.com" {
		t.Errorf("expected trycloudflare URL, got %q", url)
	}

	if err := runner.Stop(); err != nil {
		t.Errorf("stopping runner: %v", err)
	}

	statuses := mgr.Statuses()
	if len(statuses) != 0 {
		t.Errorf("expected 0 statuses, got %d", len(statuses))
	}

	_ = id
}

func TestManagerStartStop(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "mock-cloudflared")
	err := os.WriteFile(script, []byte(`#!/bin/sh
echo "https://test-xyz.trycloudflare.com" >&2
sleep 60
`), 0755)
	if err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	stateFile := filepath.Join(dir, "tunnels.json")
	mgr := NewManager(stateFile, nil)

	id := TunnelID{Project: "testproj", Service: "web"}

	runner := &Runner{cfg: RunnerConfig{
		Upstream:        "http://localhost:8080",
		CloudflaredPath: script,
	}}

	if err := runner.Start(); err != nil {
		t.Fatalf("starting runner: %v", err)
	}

	url := runner.URL()
	if url != "https://test-xyz.trycloudflare.com" {
		t.Errorf("expected trycloudflare URL from stderr, got %q", url)
	}

	select {
	case <-runner.Done():
		t.Error("runner should still be running")
	case <-time.After(100 * time.Millisecond):
	}

	if err := runner.Stop(); err != nil {
		t.Errorf("stopping runner: %v", err)
	}

	select {
	case <-runner.Done():
	case <-time.After(5 * time.Second):
		t.Error("runner did not stop in time")
	}

	_ = id
	_ = mgr
}

func TestManagerStopAll(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "tunnels.json")
	mgr := NewManager(stateFile, nil)

	if err := mgr.StopAll(); err != nil {
		t.Errorf("StopAll on empty manager: %v", err)
	}
}

type mockTokenResolver struct {
	token string
	err   error
	calls []mockResolveCall
}

type mockResolveCall struct {
	apiToken   string
	accountID  string
	tunnelName string
}

func (m *mockTokenResolver) ResolveTunnelToken(apiToken, accountID, tunnelName string) (string, error) {
	m.calls = append(m.calls, mockResolveCall{apiToken, accountID, tunnelName})
	return m.token, m.err
}

func TestStartTunnel_ResolvesJWT(t *testing.T) {
	dir := t.TempDir()
	resolver := &mockTokenResolver{token: "resolved-jwt-token"}

	stateFile := filepath.Join(dir, "tunnels.json")
	mgr := NewManager(stateFile, nil)
	mgr.tokenResolver = resolver

	// StartTunnel may fail because cloudflared isn't available or the mock
	// script doesn't behave like a real named tunnel. We only care that the
	// resolver was called with the right arguments.
	id := TunnelID{Project: "myapp", Service: "web"}
	_ = mgr.StartTunnel(id, "http://localhost:3000", "my-tunnel", "api-token", "acc-123")

	if len(resolver.calls) != 1 {
		t.Fatalf("expected 1 resolver call, got %d", len(resolver.calls))
	}
	call := resolver.calls[0]
	if call.apiToken != "api-token" {
		t.Errorf("expected apiToken %q, got %q", "api-token", call.apiToken)
	}
	if call.accountID != "acc-123" {
		t.Errorf("expected accountID %q, got %q", "acc-123", call.accountID)
	}
	if call.tunnelName != "my-tunnel" {
		t.Errorf("expected tunnelName %q, got %q", "my-tunnel", call.tunnelName)
	}

	_ = mgr.StopAll()
}

func TestStartTunnel_ResolutionError(t *testing.T) {
	dir := t.TempDir()
	resolver := &mockTokenResolver{err: fmt.Errorf("tunnel %q not found in account", "bad-tunnel")}

	stateFile := filepath.Join(dir, "tunnels.json")
	mgr := NewManager(stateFile, nil)
	mgr.tokenResolver = resolver

	id := TunnelID{Project: "myapp", Service: "web"}
	err := mgr.StartTunnel(id, "http://localhost:3000", "bad-tunnel", "api-token", "")
	if err == nil {
		t.Fatal("expected error from resolver")
	}
	if got := err.Error(); !strings.Contains(got, "not found in account") {
		t.Errorf("expected 'not found' error, got: %s", got)
	}
}

func TestStartTunnel_QuickTunnelSkipsResolver(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "cloudflared")
	err := os.WriteFile(script, []byte(`#!/bin/sh
echo "https://test-abc.trycloudflare.com"
sleep 60
`), 0755)
	if err != nil {
		t.Fatalf("writing mock script: %v", err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	resolver := &mockTokenResolver{token: "should-not-be-called"}

	stateFile := filepath.Join(dir, "tunnels.json")
	mgr := NewManager(stateFile, nil)
	mgr.tokenResolver = resolver

	id := TunnelID{Project: "myapp", Service: "web"}
	err = mgr.StartTunnel(id, "http://localhost:3000", "true", "some-token", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolver.calls) != 0 {
		t.Errorf("resolver should not be called for quick tunnels, got %d calls", len(resolver.calls))
	}

	_ = mgr.StopAll()
}

func TestStartTunnel_NamedTunnelNoToken(t *testing.T) {
	dir := t.TempDir()
	resolver := &mockTokenResolver{token: "should-not-be-called"}

	stateFile := filepath.Join(dir, "tunnels.json")
	mgr := NewManager(stateFile, nil)
	mgr.tokenResolver = resolver

	id := TunnelID{Project: "myapp", Service: "web"}
	// Named tunnel without token: should skip resolution.
	// May fail because cloudflared isn't available, but resolver should not be called.
	_ = mgr.StartTunnel(id, "http://localhost:3000", "my-tunnel", "", "")

	if len(resolver.calls) != 0 {
		t.Errorf("resolver should not be called when token is empty, got %d calls", len(resolver.calls))
	}

	_ = mgr.StopAll()
}

func TestBuildDesired_IncludesAccountID(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "tunnels.json")
	mgr := NewManager(stateFile, nil)

	cfg := config.Config{
		Version: 1,
		Settings: config.Settings{
			TLD:                 "test",
			CloudflareToken:     "global-token",
			CloudflareAccountID: "abcdef0123456789abcdef0123456789",
		},
		Projects: map[string]config.Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/tmp/myapp",
				Enabled: true,
				Services: map[string]config.Service{
					"web": {
						Proxy:  "http://localhost:3000",
						Tunnel: "my-tunnel",
					},
				},
			},
		},
	}

	desired := mgr.buildDesired(cfg)
	id := TunnelID{Project: "myapp", Service: "web"}
	d, ok := desired[id]
	if !ok {
		t.Fatal("expected desired tunnel for myapp/web")
	}
	if d.accountID != "abcdef0123456789abcdef0123456789" {
		t.Errorf("expected accountID from settings, got %q", d.accountID)
	}
	if d.token != "global-token" {
		t.Errorf("expected global token, got %q", d.token)
	}
}
