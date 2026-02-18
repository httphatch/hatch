package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeProjectConfig_Add(t *testing.T) {
	cfg := DefaultConfig()
	pc := ProjectConfig{
		Domain:   "myapp.test",
		Services: map[string]Service{"web": {Proxy: "http://localhost:3000"}},
	}

	if err := MergeProjectConfig(&cfg, "myapp", "/tmp/myapp", pc); err != nil {
		t.Fatalf("MergeProjectConfig: %v", err)
	}

	p, ok := cfg.Projects["myapp"]
	if !ok {
		t.Fatal("project not found after merge")
	}
	if p.Domain != "" {
		t.Errorf("domain should not be stored, got %q", p.Domain)
	}
	if p.Path != "/tmp/myapp" {
		t.Errorf("path: got %q, want %q", p.Path, "/tmp/myapp")
	}
	if !p.Enabled {
		t.Error("expected enabled to be true")
	}
}

func TestMergeProjectConfig_Update(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Projects["myapp"] = Project{
		Path: "/tmp/myapp", Enabled: true,
	}

	pc := ProjectConfig{
		Domain:   "newdomain.test",
		Services: map[string]Service{"web": {Proxy: "http://localhost:4000"}},
	}

	if err := MergeProjectConfig(&cfg, "myapp", "/tmp/myapp", pc); err != nil {
		t.Fatalf("MergeProjectConfig update: %v", err)
	}

	p := cfg.Projects["myapp"]
	if p.Domain != "" {
		t.Errorf("domain should not be stored, got %q", p.Domain)
	}
	if len(p.Services) != 0 {
		t.Errorf("expected no services in merged config, got %d", len(p.Services))
	}
}

func TestMergeProjectConfig_DomainConflict(t *testing.T) {
	// Create a temp directory with a hatch.yml for the existing project
	existingDir := t.TempDir()
	hatchYml := "domain: app.test\nservices:\n  web:\n    proxy: http://localhost:3000\n"
	if err := os.WriteFile(filepath.Join(existingDir, "hatch.yml"), []byte(hatchYml), 0o644); err != nil {
		t.Fatalf("writing hatch.yml: %v", err)
	}

	cfg := DefaultConfig()
	cfg.Projects["existing"] = Project{
		Path: existingDir, Enabled: true,
	}

	pc := ProjectConfig{
		Domain:   "app.test",
		Services: map[string]Service{"web": {Proxy: "http://localhost:4000"}},
	}

	err := MergeProjectConfig(&cfg, "newproj", "/tmp/new", pc)
	if err == nil {
		t.Fatal("expected domain conflict error")
	}
}

func TestMergeProjectConfig_SameProjectSameDomain(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Projects["myapp"] = Project{
		Path: "/tmp/myapp", Enabled: true,
	}

	pc := ProjectConfig{
		Domain:   "myapp.test",
		Services: map[string]Service{"api": {Proxy: "http://localhost:8000"}},
	}

	if err := MergeProjectConfig(&cfg, "myapp", "/tmp/myapp", pc); err != nil {
		t.Fatalf("re-linking same project should not conflict: %v", err)
	}
}

func TestUnmergeProject(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Projects["myapp"] = Project{
		Path: "/tmp/myapp", Enabled: true,
	}

	if err := UnmergeProject(&cfg, "myapp"); err != nil {
		t.Fatalf("UnmergeProject: %v", err)
	}

	if _, ok := cfg.Projects["myapp"]; ok {
		t.Error("project should have been removed")
	}
}

func TestUnmergeProject_NotFound(t *testing.T) {
	cfg := DefaultConfig()
	err := UnmergeProject(&cfg, "nope")
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}
