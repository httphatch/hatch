package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Load reads and validates the config from ConfigFile().
func Load() (Config, error) {
	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		msgs := FormatYAMLError(err)
		errs := make([]error, len(msgs))
		for i, m := range msgs {
			errs[i] = fmt.Errorf("%s", m)
		}
		return Config{}, fmt.Errorf("parsing config: %w", &ValidationErrors{Errs: errs})
	}

	if errs := Validate(cfg); len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid config: %w", &ValidationErrors{Errs: errs})
	}

	return cfg, nil
}

// Save atomically writes cfg to ConfigFile().
// It backs up the existing config (if any) to config.yml.bak,
// then writes to a temp file and renames, preventing partial reads.
func Save(cfg Config) error {
	path := ConfigFile()

	if err := backupConfig(path); err != nil {
		return fmt.Errorf("backing up config: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	tmp := path + ".tmp"

	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing temp config: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp) // best-effort cleanup
		return fmt.Errorf("renaming temp config: %w", err)
	}

	return nil
}

// backupConfig copies the existing config file to path.bak.
// It is a no-op if the source file does not exist.
func backupConfig(path string) error {
	src, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = src.Close() }()

	bakPath := path + ".bak"
	dst, err := os.OpenFile(bakPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(bakPath)
		return err
	}
	return dst.Close()
}

// LoadProjectConfig reads a per-project hatch.yml file.
func LoadProjectConfig(path string) (ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectConfig{}, fmt.Errorf("reading project config: %w", err)
	}

	var pc ProjectConfig
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return ProjectConfig{}, fmt.Errorf("parsing project config: %w", err)
	}

	if pc.Domain == "" {
		return ProjectConfig{}, fmt.Errorf("project config: domain is required")
	}
	if len(pc.Services) == 0 {
		return ProjectConfig{}, fmt.Errorf("project config: at least one service is required")
	}

	return pc, nil
}

// MergeAllProjectConfigs re-reads hatch.yml from each linked project's Path
// directory and overwrites the project's Services and Domain in-place.
// Projects without a hatch.yml or with an invalid one are logged and skipped.
func MergeAllProjectConfigs(cfg *Config) {
	for name, proj := range cfg.Projects {
		if proj.Path == "" {
			continue
		}
		hatchFile := filepath.Join(proj.Path, "hatch.yml")
		pc, err := LoadProjectConfig(hatchFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			log.Warn().Str("project", name).Err(err).Msg("skipping project config merge")
			continue
		}
		proj.Domain = pc.Domain
		proj.Services = pc.Services
		cfg.Projects[name] = proj
	}
}

// LoadWithProjectConfigs loads the central config, merges hatch.yml from
// each linked project, then validates. This makes project-local files the
// source of truth for services and domain.
func LoadWithProjectConfigs() (Config, error) {
	cfg, err := LoadRaw()
	if err != nil {
		return Config{}, err
	}
	MergeAllProjectConfigs(&cfg)

	if errs := Validate(cfg); len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid config: %w", &ValidationErrors{Errs: errs})
	}
	return cfg, nil
}

// LoadRaw reads the config file without validation, useful for merging.
func LoadRaw() (Config, error) {
	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// EnsureConfigDir creates the config directory and subdirectories.
func EnsureConfigDir() error {
	dirs := []string{Dir(), CertsDir(), LogsDir()}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}
	return nil
}

// EnsureConfigFile creates a default config file if one doesn't exist.
func EnsureConfigFile() error {
	path := ConfigFile()
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}

	cfg := DefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling default config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// Init ensures the config directory structure and default config file exist.
func Init() error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}
	return EnsureConfigFile()
}

// ConfigFileDir returns the directory containing the config file,
// used by the watcher to watch the directory instead of the file.
func ConfigFileDir() string {
	return filepath.Dir(ConfigFile())
}
