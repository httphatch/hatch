// Package caddy translates Hatch configuration into Caddy's JSON
// configuration format and provides a client for the Caddy admin API.
package caddy

import (
	"os"
	"path/filepath"

	"github.com/httphatch/hatch/internal/config"
)

// DefaultAdminAddr is the default Caddy admin API listen address.
const DefaultAdminAddr = "localhost:2019"

// ServerConfig holds the settings needed to run the embedded Caddy server.
// It is decoupled from config.Settings — the caller maps between them.
type ServerConfig struct {
	AdminAddr string // e.g. "localhost:2019"
}

// DataDir returns Caddy's isolated data directory under ~/.hatch/caddy/.
func DataDir() string {
	return config.CaddyDir()
}

// ConfigureDataDir sets XDG_DATA_HOME so that Caddy's internal AppDataDir()
// resolves to ~/.hatch/caddy/ instead of ~/Library/Application Support/Caddy/.
// This is needed in addition to the storage.root JSON config because Caddy's
// PKI module stores authorities under AppDataDir(), not under storage.root.
// Must be called before caddyv2.Run() and before any concurrent goroutines.
func ConfigureDataDir() error {
	return os.Setenv("XDG_DATA_HOME", config.Dir())
}

// ClearPKICache removes Caddy's cached PKI authority and issued certificates
// for the "hatch" CA. This forces Caddy to use the intermediate CA we provide
// in the PKI config and re-issue leaf certificates signed by it.
func ClearPKICache() error {
	dataDir := DataDir()
	dirs := []string{
		filepath.Join(dataDir, "pki", "authorities", "hatch"),
		filepath.Join(dataDir, "certificates", "hatch"),
	}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	return nil
}
