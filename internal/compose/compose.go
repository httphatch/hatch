package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// DiscoveredService represents a service found in a docker-compose file.
type DiscoveredService struct {
	Name   string // service name (prefixed with dir name for child dirs)
	Port   int    // host port
	Source string // relative path to compose file
}

// composeFile is a minimal representation of a docker-compose file.
type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Ports []any `yaml:"ports"`
}

var composeFileNames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// Discover scans dir and its immediate child directories for docker-compose
// files and returns the services with exposed ports.
func Discover(dir string) ([]DiscoveredService, error) {
	var all []DiscoveredService
	seen := make(map[string]bool)

	// Search in dir itself
	for _, name := range composeFileNames {
		path := filepath.Join(dir, name)
		services, err := parseComposeFile(path)
		if err != nil {
			continue
		}
		for _, s := range services {
			s.Source = name
			if !seen[s.Name] {
				seen[s.Name] = true
				all = append(all, s)
			}
		}
		break // use first match in root dir
	}

	// Search immediate child directories
	entries, err := os.ReadDir(dir)
	if err != nil {
		return all, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		childDir := filepath.Join(dir, entry.Name())
		for _, name := range composeFileNames {
			path := filepath.Join(childDir, name)
			services, err := parseComposeFile(path)
			if err != nil {
				continue
			}
			for _, s := range services {
				s.Source = filepath.Join(entry.Name(), name)
				// Prefix with dir name for deduplication
				s.Name = entry.Name()
				if !seen[s.Name] {
					seen[s.Name] = true
					all = append(all, s)
				}
			}
			break // use first match in child dir
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	return all, nil
}

func parseComposeFile(path string) ([]DiscoveredService, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	var services []DiscoveredService
	for name, svc := range cf.Services {
		for _, raw := range svc.Ports {
			port, ok := extractHostPort(raw)
			if ok {
				services = append(services, DiscoveredService{
					Name: name,
					Port: port,
				})
				break // use first valid port
			}
		}
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services, nil
}

// extractHostPort extracts the host port from a docker-compose port mapping.
// Supports:
//   - "3000"          → 3000 (container-only, treat as host port too)
//   - "8080:80"       → 8080
//   - "127.0.0.1:3000:3000" → 3000
//   - "3000-3005:3000-3005"  → skip (ranges)
//   - map with "published" key
func extractHostPort(raw any) (int, bool) {
	switch v := raw.(type) {
	case string:
		return extractHostPortFromString(v)
	case int:
		return v, v > 0
	case map[string]any:
		pub, ok := v["published"]
		if !ok {
			return 0, false
		}
		switch p := pub.(type) {
		case int:
			return p, p > 0
		case string:
			port, err := strconv.Atoi(p)
			return port, err == nil && port > 0
		}
	}
	return 0, false
}

func extractHostPortFromString(s string) (int, bool) {
	if strings.Contains(s, "-") {
		return 0, false
	}

	// Remove protocol suffix (e.g., "8080:80/tcp")
	if idx := strings.Index(s, "/"); idx != -1 {
		s = s[:idx]
	}

	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		// "3000" — container-only, treat as host port
		port, err := strconv.Atoi(parts[0])
		return port, err == nil && port > 0
	case 2:
		// "8080:80"
		port, err := strconv.Atoi(parts[0])
		return port, err == nil && port > 0
	case 3:
		// "127.0.0.1:3000:3000"
		port, err := strconv.Atoi(parts[1])
		return port, err == nil && port > 0
	}
	return 0, false
}
