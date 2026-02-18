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
		path     string
		errSubst string
	}{
		{"empty domain (no path)", "", "", "domain is required"},
		{"wrong tld", "app.com", "/tmp/myapp", "must be a valid hostname ending with .test"},
		{"bare tld", ".test", "/tmp/myapp", "must be a valid hostname ending with .test"},
		{"invalid label", "app-.test", "/tmp/myapp", "must be a valid hostname ending with .test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			p := cfg.Projects["myapp"]
			p.Domain = tt.domain
			p.Path = tt.path
			cfg.Projects["myapp"] = p
			errs := Validate(cfg)
			requireError(t, errs, tt.errSubst)
		})
	}

	// Linked projects (with path) may have empty domain in config.yml;
	// domain comes from .hatch.yml at load time.
	t.Run("empty domain with path (linked project)", func(t *testing.T) {
		cfg := validConfig()
		p := cfg.Projects["myapp"]
		p.Domain = ""
		cfg.Projects["myapp"] = p
		errs := Validate(cfg)
		for _, e := range errs {
			if e.Error() == "projects.myapp.domain is required" {
				t.Error("linked project with empty domain should not fail validation")
			}
		}
	})
}

func TestValidate_ProjectPath(t *testing.T) {
	cfg := validConfig()
	p := cfg.Projects["myapp"]
	p.Path = ""
	cfg.Projects["myapp"] = p
	errs := Validate(cfg)
	requireError(t, errs, "path is required")
}

func TestValidate_ProjectServicesEmpty_NoPath(t *testing.T) {
	cfg := validConfig()
	p := cfg.Projects["myapp"]
	p.Services = map[string]Service{}
	p.Path = ""
	cfg.Projects["myapp"] = p
	errs := Validate(cfg)
	requireError(t, errs, "must have at least one entry")
}

func TestValidate_ProjectServicesEmpty_WithPath(t *testing.T) {
	cfg := validConfig()
	p := cfg.Projects["myapp"]
	p.Services = map[string]Service{}
	cfg.Projects["myapp"] = p
	errs := Validate(cfg)
	if len(errs) != 0 {
		t.Errorf("linked project with path should allow empty services, got %v", errs)
	}
}

func TestValidate_ServiceProxy(t *testing.T) {
	tests := []struct {
		name     string
		proxy    string
		errSubst string
	}{
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

func TestValidate_ServiceCommandOrProxy(t *testing.T) {
	tests := []struct {
		name     string
		svc      Service
		wantErr  string
	}{
		{
			name:    "neither command nor proxy",
			svc:     Service{},
			wantErr: "command or proxy is required",
		},
		{
			name: "command only",
			svc:  Service{Command: "php artisan queue:work"},
		},
		{
			name: "proxy only",
			svc:  Service{Proxy: "http://localhost:3000"},
		},
		{
			name: "command and proxy",
			svc:  Service{Command: "php artisan serve --port=8000", Proxy: "http://localhost:8000"},
		},
		{
			name:    "route without proxy",
			svc:     Service{Command: "echo hello", Route: "/api/*"},
			wantErr: "route requires proxy",
		},
		{
			name:    "subdomain without proxy",
			svc:     Service{Command: "echo hello", Subdomain: "api"},
			wantErr: "subdomain requires proxy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			p := cfg.Projects["myapp"]
			p.Services = map[string]Service{"web": tt.svc}
			cfg.Projects["myapp"] = p
			errs := Validate(cfg)
			if tt.wantErr != "" {
				requireError(t, errs, tt.wantErr)
			} else if len(errs) != 0 {
				t.Errorf("expected no errors, got %v", errs)
			}
		})
	}
}

func TestValidate_ServiceRoute(t *testing.T) {
	tests := []struct {
		name    string
		route   string
		wantErr string
	}{
		{"valid route", "/api", ""},
		{"valid wildcard", "/api/*", ""},
		{"no leading slash", "api", "must begin with '/'"},
		{"newline", "/api\n/evil", "contains invalid characters"},
		{"carriage return", "/api\r\n", "contains invalid characters"},
		{"null byte", "/api\x00", "contains invalid characters"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			p := cfg.Projects["myapp"]
			p.Services = map[string]Service{"web": {Proxy: "http://localhost:3000", Route: tt.route}}
			cfg.Projects["myapp"] = p
			errs := Validate(cfg)
			if tt.wantErr != "" {
				requireError(t, errs, tt.wantErr)
			} else if len(errs) != 0 {
				t.Errorf("expected no errors, got %v", errs)
			}
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

func TestValidate_ServiceEnvFile(t *testing.T) {
	tests := []struct {
		name    string
		envFile string
		wantErr string
	}{
		{"relative path", ".env", ""},
		{"nested relative", "config/.env", ""},
		{"absolute path", "/etc/secrets/.env", "must be a relative path"},
		{"parent traversal", "../.env", "must not contain '..'"},
		{"nested traversal", "config/../../.env", "must not contain '..'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			p := cfg.Projects["myapp"]
			p.Services = map[string]Service{"web": {Command: "npm start", EnvFile: tt.envFile}}
			cfg.Projects["myapp"] = p
			errs := Validate(cfg)
			if tt.wantErr != "" {
				requireError(t, errs, tt.wantErr)
			} else if len(errs) != 0 {
				t.Errorf("expected no errors, got %v", errs)
			}
		})
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
