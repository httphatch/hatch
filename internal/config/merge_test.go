package config

import "testing"

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
	if p.Domain != "myapp.test" {
		t.Errorf("domain: got %q, want %q", p.Domain, "myapp.test")
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
		Domain: "myapp.test", Path: "/tmp/myapp", Enabled: true,
		Services: map[string]Service{"web": {Proxy: "http://localhost:3000"}},
	}

	pc := ProjectConfig{
		Domain:   "myapp.test",
		Services: map[string]Service{"web": {Proxy: "http://localhost:4000"}},
	}

	if err := MergeProjectConfig(&cfg, "myapp", "/tmp/myapp", pc); err != nil {
		t.Fatalf("MergeProjectConfig update: %v", err)
	}

	if cfg.Projects["myapp"].Services["web"].Proxy != "http://localhost:4000" {
		t.Error("proxy should have been updated")
	}
}

func TestMergeProjectConfig_DomainConflict(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Projects["existing"] = Project{
		Domain: "app.test", Path: "/tmp/existing", Enabled: true,
		Services: map[string]Service{"web": {Proxy: "http://localhost:3000"}},
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
		Domain: "myapp.test", Path: "/tmp/myapp", Enabled: true,
		Services: map[string]Service{"web": {Proxy: "http://localhost:3000"}},
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
		Domain: "myapp.test", Path: "/tmp/myapp", Enabled: true,
		Services: map[string]Service{"web": {Proxy: "http://localhost:3000"}},
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
