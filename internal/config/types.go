package config

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Config is the top-level Hatch configuration.
type Config struct {
	Version  int                `yaml:"version" json:"version"`
	Settings Settings           `yaml:"settings" json:"settings"`
	Projects map[string]Project `yaml:"projects" json:"projects"`
}

// Settings holds global Hatch settings.
type Settings struct {
	TLD                 string `yaml:"tld" json:"tld"`
	HTTPPort            int    `yaml:"http_port" json:"http_port"`
	HTTPSPort           int    `yaml:"https_port" json:"https_port"`
	AutoStart           bool   `yaml:"auto_start" json:"auto_start"`
	TrayIcon            bool   `yaml:"tray_icon" json:"tray_icon"`
	LogLevel            string `yaml:"log_level" json:"log_level"`
	CloudflareToken     string `yaml:"cloudflare_token,omitempty" json:"cloudflare_token,omitempty"`
	CloudflareAccountID string `yaml:"cloudflare_account_id,omitempty" json:"cloudflare_account_id,omitempty"`
}

// Project defines a single project's proxy configuration.
type Project struct {
	Domain          string             `yaml:"domain,omitempty" json:"domain"`
	Path            string             `yaml:"path" json:"path"`
	Enabled         bool               `yaml:"enabled" json:"enabled"`
	Services        map[string]Service `yaml:"services,omitempty" json:"services"`
	CloudflareToken string             `yaml:"cloudflare_token,omitempty" json:"cloudflare_token,omitempty"`
}

// TunnelValue holds the tunnel configuration for a service.
// It accepts both boolean and string YAML values.
type TunnelValue string

// UnmarshalYAML handles both boolean and string YAML values for tunnel config.
func (t *TunnelValue) UnmarshalYAML(value *yaml.Node) error {
	if value.Tag == "!!bool" {
		if value.Value == "true" {
			*t = "true"
		}
		// tunnel: false means no tunnel (leave as zero value "").
		return nil
	}
	*t = TunnelValue(value.Value)
	return nil
}

// UnmarshalJSON handles both boolean and string JSON values for tunnel config.
func (t *TunnelValue) UnmarshalJSON(data []byte) error {
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		if b {
			*t = "true"
		}
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("tunnel must be a string or boolean: %w", err)
	}
	*t = TunnelValue(s)
	return nil
}

// Service defines how a single service is proxied and optionally managed.
type Service struct {
	Proxy     string      `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	Command   string      `yaml:"command,omitempty" json:"command,omitempty"`
	Dir       string      `yaml:"dir,omitempty" json:"dir,omitempty"`
	Route     string      `yaml:"route,omitempty" json:"route,omitempty"`
	Subdomain string      `yaml:"subdomain,omitempty" json:"subdomain,omitempty"`
	WebSocket bool        `yaml:"websocket,omitempty" json:"websocket,omitempty"`
	EnvFile   string      `yaml:"env_file,omitempty" json:"env_file,omitempty"`
	Tunnel    TunnelValue `yaml:"tunnel,omitempty" json:"tunnel,omitempty"`
}

// ProjectConfig is the schema for a per-project hatch.yml file.
type ProjectConfig struct {
	Domain          string             `yaml:"domain"`
	Services        map[string]Service `yaml:"services"`
	CloudflareToken string             `yaml:"cloudflare_token,omitempty"`
}
