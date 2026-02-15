package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestWatcher_CallbackOnChange(t *testing.T) {
	home := setupTestHome(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var received *Config

	w, err := NewWatcher(func(cfg Config) {
		mu.Lock()
		defer mu.Unlock()
		received = &cfg
	})
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	// Modify the config
	cfg := DefaultConfig()
	cfg.Settings.LogLevel = "debug"
	data, _ := yaml.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(home, configFileName), data, 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for debounce + processing
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := received
		mu.Unlock()
		if got != nil {
			if got.Settings.LogLevel != "debug" {
				t.Errorf("log level: got %q, want %q", got.Settings.LogLevel, "debug")
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("callback was not invoked within timeout")
}

func TestWatcher_InvalidConfigSkipped(t *testing.T) {
	home := setupTestHome(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	callCount := 0
	var mu sync.Mutex

	w, err := NewWatcher(func(cfg Config) {
		mu.Lock()
		defer mu.Unlock()
		callCount++
	})
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	// Write invalid config
	if err := os.WriteFile(filepath.Join(home, configFileName), []byte("version: 0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait enough time for debounce to fire
	time.Sleep(1500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if callCount != 0 {
		t.Errorf("callback should not have been called for invalid config, got %d calls", callCount)
	}
}

func TestWatcher_CleanShutdown(t *testing.T) {
	setupTestHome(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(func(cfg Config) {})
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}

	// Close should not hang or panic
	done := make(chan struct{})
	go func() {
		_ = w.Close()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(3 * time.Second):
		t.Fatal("Close did not return within timeout")
	}
}
