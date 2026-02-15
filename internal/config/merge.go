package config

import "fmt"

// MergeProjectConfig adds or updates a project in the config from a ProjectConfig.
// The name is the project key (e.g. "myapp"), and projectPath is the filesystem path.
func MergeProjectConfig(cfg *Config, name string, projectPath string, pc ProjectConfig) error {
	// Check for domain conflicts with other projects
	for existingName, existingProj := range cfg.Projects {
		if existingName == name {
			continue // same project, allow update
		}
		if existingProj.Domain == pc.Domain {
			return fmt.Errorf("domain %q is already used by project %q", pc.Domain, existingName)
		}
	}

	cfg.Projects[name] = Project{
		Domain:   pc.Domain,
		Path:     projectPath,
		Enabled:  true,
		Services: pc.Services,
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
