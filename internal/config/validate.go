package config

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

var validHostnameLabel = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
var validTunnelName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)
var validCloudflareAccountID = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)
var validTokenChars = regexp.MustCompile(`^[\x21-\x7E]+$`)

var allowedTLDs = map[string]bool{
	"test":      true,
	"localhost": true,
	"local":     true,
	"dev":       true,
}

var allowedLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

// Validate checks cfg for structural and semantic correctness.
// It returns a slice of all validation errors found (never partial).
func Validate(cfg Config) []error {
	var errs []error

	// Version
	if cfg.Version != 1 {
		errs = append(errs, fmt.Errorf("version must be 1, got %d", cfg.Version))
	}

	// Settings
	errs = append(errs, validateSettings(cfg.Settings)...)

	// Projects
	domains := make(map[string]string) // domain -> project name
	for name, proj := range cfg.Projects {
		errs = append(errs, validateProject(name, proj, cfg.Settings, domains)...)
	}

	return errs
}

func validateSettings(s Settings) []error {
	var errs []error

	if !allowedTLDs[s.TLD] {
		errs = append(errs, fmt.Errorf("settings.tld must be one of: test, localhost, local, dev; got %q", s.TLD))
	}

	if s.HTTPPort < 1 || s.HTTPPort > 65535 {
		errs = append(errs, fmt.Errorf("settings.http_port must be 1-65535, got %d", s.HTTPPort))
	}

	if s.HTTPSPort < 1 || s.HTTPSPort > 65535 {
		errs = append(errs, fmt.Errorf("settings.https_port must be 1-65535, got %d", s.HTTPSPort))
	}

	if s.HTTPPort >= 1 && s.HTTPPort <= 65535 && s.HTTPSPort >= 1 && s.HTTPSPort <= 65535 && s.HTTPPort == s.HTTPSPort {
		errs = append(errs, fmt.Errorf("settings.http_port and settings.https_port must differ, both are %d", s.HTTPPort))
	}

	if !allowedLogLevels[s.LogLevel] {
		errs = append(errs, fmt.Errorf("settings.log_level must be one of: debug, info, warn, error; got %q", s.LogLevel))
	}

	if len(s.CloudflareToken) > 512 {
		errs = append(errs, fmt.Errorf("settings.cloudflare_token must not exceed 512 characters"))
	}
	if s.CloudflareToken != "" && !validTokenChars.MatchString(s.CloudflareToken) {
		errs = append(errs, fmt.Errorf("settings.cloudflare_token must contain only printable ASCII characters"))
	}

	if s.CloudflareAccountID != "" && !validCloudflareAccountID.MatchString(s.CloudflareAccountID) {
		errs = append(errs, fmt.Errorf("settings.cloudflare_account_id must be a 32-character hex string"))
	}

	if s.SessionPortMin != 0 && (s.SessionPortMin < 1024 || s.SessionPortMin > 65535) {
		errs = append(errs, fmt.Errorf("settings.session_port_min must be 1024-65535, got %d", s.SessionPortMin))
	}
	if s.SessionPortMax != 0 && (s.SessionPortMax < 1024 || s.SessionPortMax > 65535) {
		errs = append(errs, fmt.Errorf("settings.session_port_max must be 1024-65535, got %d", s.SessionPortMax))
	}
	if s.SessionPortMin != 0 && s.SessionPortMax != 0 && s.SessionPortMin > s.SessionPortMax {
		errs = append(errs, fmt.Errorf("settings.session_port_min (%d) must not exceed session_port_max (%d)", s.SessionPortMin, s.SessionPortMax))
	}
	if s.SessionTTL < 0 {
		errs = append(errs, fmt.Errorf("settings.session_ttl must be >= 0, got %d", s.SessionTTL))
	}

	if s.APIPort != 0 && (s.APIPort < 1 || s.APIPort > 65535) {
		errs = append(errs, fmt.Errorf("settings.api_port must be 1-65535, got %d", s.APIPort))
	}
	if s.DNSPort != 0 && (s.DNSPort < 1 || s.DNSPort > 65535) {
		errs = append(errs, fmt.Errorf("settings.dns_port must be 1-65535, got %d", s.DNSPort))
	}
	if s.CaddyAdminPort != 0 && (s.CaddyAdminPort < 1 || s.CaddyAdminPort > 65535) {
		errs = append(errs, fmt.Errorf("settings.caddy_admin_port must be 1-65535, got %d", s.CaddyAdminPort))
	}

	return errs
}

func validateProject(name string, p Project, s Settings, domains map[string]string) []error {
	var errs []error
	prefix := fmt.Sprintf("projects.%s", name)
	tld := s.TLD

	// Domain: valid hostname ending with configured TLD
	switch {
	case p.Domain == "" && p.Path != "":
		// Domain lives in hatch.yml for linked projects; not required in central config.
	case p.Domain == "":
		errs = append(errs, fmt.Errorf("%s.domain is required", prefix))
	case !isValidDomain(p.Domain, tld):
		errs = append(errs, fmt.Errorf("%s.domain %q must be a valid hostname ending with .%s", prefix, p.Domain, tld))
	default:
		if other, exists := domains[p.Domain]; exists {
			errs = append(errs, fmt.Errorf("duplicate domain %q in projects %q and %q", p.Domain, other, name))
		}
		domains[p.Domain] = name
	}

	// Path
	if p.Path == "" {
		errs = append(errs, fmt.Errorf("%s.path is required", prefix))
	} else if !filepath.IsAbs(p.Path) {
		errs = append(errs, fmt.Errorf("%s.path must be an absolute path, got %q", prefix, p.Path))
	}

	// Services — linked projects (with a Path) may have services defined in
	// hatch.yml rather than in the central config, so empty services are
	// allowed when a path is set.
	if len(p.Services) == 0 && p.Path == "" {
		errs = append(errs, fmt.Errorf("%s.services must have at least one entry", prefix))
	}
	for svcName, svc := range p.Services {
		errs = append(errs, validateService(prefix, svcName, svc)...)
	}

	return errs
}

func validateService(prefix, name string, s Service) []error {
	var errs []error
	svcPrefix := fmt.Sprintf("%s.services.%s", prefix, name)

	// At least one of command or proxy is required.
	if s.Command == "" && s.Proxy == "" {
		errs = append(errs, fmt.Errorf("%s: command or proxy is required", svcPrefix))
	}

	// Tunnel validation.
	if s.Tunnel != "" && s.Proxy == "" {
		errs = append(errs, fmt.Errorf("%s.tunnel requires proxy to be set", svcPrefix))
	}
	if s.Tunnel != "" && s.Tunnel != "true" {
		if !validTunnelName.MatchString(string(s.Tunnel)) {
			errs = append(errs, fmt.Errorf("%s.tunnel %q must contain only alphanumeric characters, hyphens, or underscores", svcPrefix, string(s.Tunnel)))
		}
	}

	// Proxy URL (validated only when set, skip if it contains template variables).
	if s.Proxy != "" && !strings.Contains(s.Proxy, "{{") {
		u, err := url.Parse(s.Proxy)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			errs = append(errs, fmt.Errorf("%s.proxy %q must be a valid URL with http or https scheme", svcPrefix, s.Proxy))
		}
	}

	// Route format.
	if s.Route != "" {
		if !strings.HasPrefix(s.Route, "/") {
			errs = append(errs, fmt.Errorf("%s.route must begin with '/', got %q", svcPrefix, s.Route))
		}
		if strings.ContainsAny(s.Route, "\r\n\x00") {
			errs = append(errs, fmt.Errorf("%s.route contains invalid characters", svcPrefix))
		}
	}

	// Route and subdomain require proxy.
	if s.Route != "" && s.Proxy == "" {
		errs = append(errs, fmt.Errorf("%s.route requires proxy to be set", svcPrefix))
	}
	if s.Subdomain != "" && s.Proxy == "" {
		errs = append(errs, fmt.Errorf("%s.subdomain requires proxy to be set", svcPrefix))
	}

	// Subdomain format (optional).
	if s.Subdomain != "" && !validHostnameLabel.MatchString(s.Subdomain) {
		errs = append(errs, fmt.Errorf("%s.subdomain %q must be a valid hostname label", svcPrefix, s.Subdomain))
	}

	// Dir must be a relative path without traversal (resolved against project path).
	if s.Dir != "" {
		if filepath.IsAbs(s.Dir) {
			errs = append(errs, fmt.Errorf("%s.dir must be a relative path, got %q", svcPrefix, s.Dir))
		} else if strings.Contains(s.Dir, "..") {
			errs = append(errs, fmt.Errorf("%s.dir must not contain '..', got %q", svcPrefix, s.Dir))
		}
	}

	// Dir requires command.
	if s.Dir != "" && s.Command == "" {
		errs = append(errs, fmt.Errorf("%s.dir requires command to be set", svcPrefix))
	}

	// EnvFile must be a relative path without traversal.
	if s.EnvFile != "" {
		if filepath.IsAbs(s.EnvFile) {
			errs = append(errs, fmt.Errorf("%s.env_file must be a relative path, got %q", svcPrefix, s.EnvFile))
		} else if strings.Contains(s.EnvFile, "..") {
			errs = append(errs, fmt.Errorf("%s.env_file must not contain '..', got %q", svcPrefix, s.EnvFile))
		}
	}

	return errs
}

// isValidDomain checks that domain is a valid hostname ending with .<tld>.
func isValidDomain(domain, tld string) bool {
	suffix := "." + tld
	if !strings.HasSuffix(domain, suffix) {
		return false
	}

	base := strings.TrimSuffix(domain, suffix)
	if base == "" {
		return false
	}

	// Check each label
	labels := strings.Split(base, ".")
	for _, label := range labels {
		if !validHostnameLabel.MatchString(label) {
			return false
		}
	}

	return true
}
