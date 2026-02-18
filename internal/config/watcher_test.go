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

	cfg := DefaultConfig()
	w, err := NewWatcher(cfg, func(cfg Config) {
		mu.Lock()
		defer mu.Unlock()
		received = &cfg
	})
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	// Modify the config
	cfg.Settings.LogLevel = "debug"
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
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

	cfg := DefaultConfig()
	w, err := NewWatcher(cfg, func(cfg Config) {
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

	cfg := DefaultConfig()
	w, err := NewWatcher(cfg, func(cfg Config) {})
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

func TestWatcher_ProjectDirChange(t *testing.T) {
	home := setupTestHome(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// Create a project directory with hatch.yml
	projDir := t.TempDir()
	hatchYml := filepath.Join(projDir, "hatch.yml")
	if err := os.WriteFile(hatchYml, []byte("domain: myapp.test\nservices:\n  web:\n    proxy: http://localhost:3000\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Save config with a project pointing to projDir
	cfg := DefaultConfig()
	cfg.Projects["myapp"] = Project{
		Domain:  "myapp.test",
		Path:    projDir,
		Enabled: true,
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, configFileName), data, 0644); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var received *Config

	w, err := NewWatcher(cfg, func(cfg Config) {
		mu.Lock()
		defer mu.Unlock()
		received = &cfg
	})
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	// Modify hatch.yml in project dir
	if err := os.WriteFile(hatchYml, []byte("domain: myapp.test\nservices:\n  web:\n    proxy: http://localhost:4000\n"), 0644); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := received
		mu.Unlock()
		if got != nil {
			proj, ok := got.Projects["myapp"]
			if !ok {
				t.Fatal("project myapp not found after reload")
			}
			if svc, ok := proj.Services["web"]; ok {
				if svc.Proxy == "http://localhost:4000" {
					return
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("callback was not invoked with updated project config within timeout")
}
