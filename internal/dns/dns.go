// Package dns manages local DNS resolution for development TLDs,
// embedding a lightweight DNS server that resolves wildcard queries
// for the configured TLD and forwards everything else upstream.
package dns

const (
	// DefaultPort is the default port for the embedded DNS server.
	// Port 5053 avoids needing root to bind.
	DefaultPort = 5053

	// DefaultListenIP is the default IP address for the DNS server.
	DefaultListenIP = "127.0.0.1"

	// ResolverDir is the macOS per-TLD resolver directory.
	ResolverDir = "/etc/resolver"
)

// ServerConfig holds the settings needed to run the embedded DNS server.
// It is decoupled from config.Settings — the caller maps between them.
type ServerConfig struct {
	TLD      string // e.g. "test"
	ListenIP string // e.g. "127.0.0.1"
	Port     int    // e.g. 5053
}
