package tunnel

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
