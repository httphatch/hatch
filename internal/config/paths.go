package config

import (
	"os"
	"path/filepath"
)

const (
	configDirName  = ".hatch"
	configFileName = "config.yml"
	certsDirName   = "certs"
	logsDirName    = "logs"
)

// Dir returns the Hatch configuration directory.
// If HATCH_HOME is set, it must be an absolute path;
// otherwise defaults to ~/.hatch.
func Dir() string {
	if v := os.Getenv("HATCH_HOME"); v != "" {
		if !filepath.IsAbs(v) {
			// Resolve relative paths to absolute to prevent traversal.
			if abs, err := filepath.Abs(v); err == nil {
				return filepath.Clean(abs)
			}
		}
		return filepath.Clean(v)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", configDirName)
	}
	return filepath.Join(home, configDirName)
}

// ConfigFile returns the path to the main config file.
func ConfigFile() string {
	return filepath.Join(Dir(), configFileName)
}

// CertsDir returns the path to the certificates directory.
func CertsDir() string {
	return filepath.Join(Dir(), certsDirName)
}

// LogsDir returns the path to the logs directory.
func LogsDir() string {
	return filepath.Join(Dir(), logsDirName)
}

// LogFile returns the path to the daemon log file.
func LogFile() string {
	return filepath.Join(LogsDir(), "hatch.log")
}

// CaddyDir returns the path to the isolated Caddy data directory.
func CaddyDir() string {
	return filepath.Join(Dir(), "caddy")
}
