package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit_CreatesDirectoryAndConfig(t *testing.T) {
	home := setupTestHome(t)

	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Config dir exists
	if _, err := os.Stat(home); err != nil {
		t.Errorf("config dir should exist: %v", err)
	}

	// Certs dir exists
	if _, err := os.Stat(filepath.Join(home, certsDirName)); err != nil {
		t.Errorf("certs dir should exist: %v", err)
	}

	// Config file exists
	if _, err := os.Stat(filepath.Join(home, configFileName)); err != nil {
		t.Errorf("config file should exist: %v", err)
	}
}

func TestInit_Idempotent(t *testing.T) {
	setupTestHome(t)

	if err := Init(); err != nil {
		t.Fatalf("Init first call: %v", err)
	}
	if err := Init(); err != nil {
		t.Fatalf("Init second call: %v", err)
	}
}

func TestInit_DefaultConfigIsValid(t *testing.T) {
	setupTestHome(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadRaw()
	if err != nil {
		t.Fatalf("LoadRaw: %v", err)
	}

	errs := Validate(cfg)
	if len(errs) != 0 {
		t.Errorf("default config should be valid, got %v", errs)
	}
}

func TestEnsureConfigFile_DoesNotOverwrite(t *testing.T) {
	home := setupTestHome(t)
	if err := EnsureConfigDir(); err != nil {
		t.Fatal(err)
	}

	// Write a custom config
	custom := []byte("version: 1\nsettings:\n  tld: dev\n  http_port: 80\n  https_port: 443\n  auto_start: false\n  log_level: debug\nprojects: {}\n")
	path := filepath.Join(home, configFileName)
	if err := os.WriteFile(path, custom, 0644); err != nil {
		t.Fatal(err)
	}

	// EnsureConfigFile should not overwrite
	if err := EnsureConfigFile(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(custom) {
		t.Error("EnsureConfigFile should not overwrite existing file")
	}
}
