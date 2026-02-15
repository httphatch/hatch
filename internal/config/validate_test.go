package config

import (
	"strings"
	"testing"
)

func validConfig() Config {
	return Config{
		Version: 1,
		Settings: Settings{
			TLD:       "test",
			HTTPPort:  80,
			HTTPSPort: 443,
			AutoStart: true,
			LogLevel:  "info",
		},
		Projects: map[string]Project{
			"myapp": {
				Domain:  "myapp.test",
				Path:    "/tmp/myapp",
				Enabled: true,
				Services: map[string]Service{
					"web": {Proxy: "http://localhost:3000"},
				},
			},
		},
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	errs := Validate(validConfig())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_Version(t *testing.T) {
	cfg := validConfig()
	cfg.Version = 0
	errs := Validate(cfg)
	requireError(t, errs, "version must be 1")

	cfg.Version = 2
	errs = Validate(cfg)
	requireError(t, errs, "version must be 1")
}

func TestValidate_TLD(t *testing.T) {
	for _, tld := range []string{"test", "localhost", "local", "dev"} {
		cfg := validConfig()
		cfg.Settings.TLD = tld
		cfg.Projects["myapp"] = Project{
			Domain: "myapp." + tld, Path: "/tmp", Enabled: true,
			Services: map[string]Service{"web": {Proxy: "http://localhost:3000"}},
		}
		errs := Validate(cfg)
		if len(errs) != 0 {
			t.Errorf("tld %q should be valid, got %v", tld, errs)
		}
	}

	cfg := validConfig()
	cfg.Settings.TLD = "com"
	errs := Validate(cfg)
	requireError(t, errs, "settings.tld must be one of")
}

func TestValidate_Ports(t *testing.T) {
	tests := []struct {
		name     string
		http     int
		https    int
		errSubst string
	}{
		{"http too low", 0, 443, "settings.http_port must be 1-65535"},
		{"http too high", 70000, 443, "settings.http_port must be 1-65535"},
		{"https too low", 80, 0, "settings.https_port must be 1-65535"},
		{"https too high", 80, 70000, "settings.https_port must be 1-65535"},
		{"same port", 443, 443, "must differ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Settings.HTTPPort = tt.http
			cfg.Settings.HTTPSPort = tt.https
			errs := Validate(cfg)
			requireError(t, errs, tt.errSubst)
		})
	}
}

func TestValidate_LogLevel(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		cfg := validConfig()
		cfg.Settings.LogLevel = level
		errs := Validate(cfg)
		if len(errs) != 0 {
			t.Errorf("log level %q should be valid, got %v", level, errs)
		}
	}

	cfg := validConfig()
	cfg.Settings.LogLevel = "trace"
	errs := Validate(cfg)
	requireError(t, errs, "settings.log_level must be one of")
}

func TestValidate_ProjectDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		errSubst string
	}{
		{"empty domain", "", "domain is required"},
		{"wrong tld", "app.com", "must be a valid hostname ending with .test"},
		{"bare tld", ".test", "must be a valid hostname ending with .test"},
		{"invalid label", "app-.test", "must be a valid hostname ending with .test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			p := cfg.Projects["myapp"]
			p.Domain = tt.domain
			cfg.Projects["myapp"] = p
			errs := Validate(cfg)
			requireError(t, errs, tt.errSubst)
		})
	}
}

func TestValidate_ProjectPath(t *testing.T) {
	cfg := validConfig()
	p := cfg.Projects["myapp"]
	p.Path = ""
	cfg.Projects["myapp"] = p
	errs := Validate(cfg)
	requireError(t, errs, "path is required")
}

func TestValidate_ProjectServicesEmpty(t *testing.T) {
	cfg := validConfig()
	p := cfg.Projects["myapp"]
	p.Services = map[string]Service{}
	cfg.Projects["myapp"] = p
	errs := Validate(cfg)
	requireError(t, errs, "must have at least one entry")
}

func TestValidate_ServiceProxy(t *testing.T) {
	tests := []struct {
		name     string
		proxy    string
		errSubst string
	}{
		{"empty", "", "proxy is required"},
		{"no scheme", "localhost:3000", "must be a valid URL"},
		{"ftp scheme", "ftp://localhost:3000", "must be a valid URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			p := cfg.Projects["myapp"]
			p.Services = map[string]Service{"web": {Proxy: tt.proxy}}
			cfg.Projects["myapp"] = p
			errs := Validate(cfg)
			requireError(t, errs, tt.errSubst)
		})
	}
}

func TestValidate_ServiceSubdomain(t *testing.T) {
	cfg := validConfig()
	p := cfg.Projects["myapp"]
	p.Services = map[string]Service{"web": {Proxy: "http://localhost:3000", Subdomain: "-bad"}}
	cfg.Projects["myapp"] = p
	errs := Validate(cfg)
	requireError(t, errs, "must be a valid hostname label")
}

func TestValidate_DuplicateDomains(t *testing.T) {
	cfg := validConfig()
	cfg.Projects["other"] = Project{
		Domain: "myapp.test", Path: "/tmp/other", Enabled: true,
		Services: map[string]Service{"web": {Proxy: "http://localhost:4000"}},
	}
	errs := Validate(cfg)
	requireError(t, errs, "duplicate domain")
}

func TestValidate_NoProjects(t *testing.T) {
	cfg := validConfig()
	cfg.Projects = map[string]Project{}
	errs := Validate(cfg)
	if len(errs) != 0 {
		t.Fatalf("config with no projects should be valid, got %v", errs)
	}
}

func requireError(t *testing.T, errs []error, substr string) {
	t.Helper()
	for _, err := range errs {
		if strings.Contains(err.Error(), substr) {
			return
		}
	}
	t.Errorf("expected error containing %q, got %v", substr, errs)
}
