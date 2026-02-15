package config

// Config is the top-level Hatch configuration.
type Config struct {
	Version  int                `yaml:"version" json:"version"`
	Settings Settings           `yaml:"settings" json:"settings"`
	Projects map[string]Project `yaml:"projects" json:"projects"`
}

// Settings holds global Hatch settings.
type Settings struct {
	TLD       string `yaml:"tld" json:"tld"`
	HTTPPort  int    `yaml:"http_port" json:"http_port"`
	HTTPSPort int    `yaml:"https_port" json:"https_port"`
	AutoStart bool   `yaml:"auto_start" json:"auto_start"`
	LogLevel  string `yaml:"log_level" json:"log_level"`
}

// Project defines a single project's proxy configuration.
type Project struct {
	Domain   string             `yaml:"domain" json:"domain"`
	Path     string             `yaml:"path" json:"path"`
	Enabled  bool               `yaml:"enabled" json:"enabled"`
	Services map[string]Service `yaml:"services" json:"services"`
}

// Service defines how a single service is proxied.
type Service struct {
	Proxy     string `yaml:"proxy" json:"proxy"`
	Route     string `yaml:"route,omitempty" json:"route,omitempty"`
	Subdomain string `yaml:"subdomain,omitempty" json:"subdomain,omitempty"`
	WebSocket bool   `yaml:"websocket,omitempty" json:"websocket,omitempty"`
}

// ProjectConfig is the schema for a per-project .hatch.yml file.
type ProjectConfig struct {
	Domain   string             `yaml:"domain"`
	Services map[string]Service `yaml:"services"`
}
