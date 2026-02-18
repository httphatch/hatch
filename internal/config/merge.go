package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// MergeProjectConfig adds or updates a project in the config from a ProjectConfig.
// The name is the project key (e.g. "myapp"), and projectPath is the filesystem path.
// Domain is not stored in config.yml; it comes from hatch.yml at load time.
func MergeProjectConfig(cfg *Config, name string, projectPath string, pc ProjectConfig) error {
	// Check for domain conflicts by loading each existing project's hatch.yml
	for existingName, existingProj := range cfg.Projects {
		if existingName == name {
			continue // same project, allow update
		}
		if existingProj.Path == "" || !filepath.IsAbs(existingProj.Path) {
			continue
		}
		hatchFile := filepath.Join(existingProj.Path, "hatch.yml")
		existingPC, err := LoadProjectConfig(hatchFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("checking domain conflicts for project %q: %w", existingName, err)
		}
		if existingPC.Domain == pc.Domain {
			return fmt.Errorf("domain %q is already used by project %q", pc.Domain, existingName)
		}
	}

	cfg.Projects[name] = Project{
		Path:    projectPath,
		Enabled: true,
	}

	return nil
}

// UnmergeProject removes a project from the config by name.
// Returns an error if the project does not exist.
func UnmergeProject(cfg *Config, name string) error {
	if _, exists := cfg.Projects[name]; !exists {
		return fmt.Errorf("project %q not found", name)
	}
	delete(cfg.Projects, name)
	return nil
}
