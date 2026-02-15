package dns

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// defaultUpstreams are used when system DNS discovery fails or returns
// no usable servers.
var defaultUpstreams = []string{"8.8.8.8:53", "1.1.1.1:53"}

// SystemDNSServers discovers the system's configured DNS servers by
// parsing the output of "scutil --dns". It filters out loopback
// addresses to avoid forwarding loops. If discovery fails or returns
// no servers, it falls back to defaultUpstreams.
func SystemDNSServers() ([]string, error) {
	out, err := exec.Command("scutil", "--dns").Output()
	if err != nil {
		return defaultUpstreams, fmt.Errorf("running scutil --dns: %w", err)
	}

	servers := parseScutilDNS(string(out))
	if len(servers) == 0 {
		return defaultUpstreams, nil
	}
	return servers, nil
}

// parseScutilDNS extracts nameserver addresses from scutil --dns
// output. It returns each unique nameserver as "ip:53", filtering out
// loopback addresses (127.0.0.0/8) to prevent forwarding loops.
func parseScutilDNS(output string) []string {
	seen := make(map[string]bool)
	var servers []string

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "nameserver[") {
			continue
		}

		// Format: "nameserver[0] : 192.168.1.1"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		ip := strings.TrimSpace(parts[1])
		if ip == "" {
			continue
		}

		// Filter out loopback addresses to avoid forwarding loops.
		parsed := net.ParseIP(ip)
		if parsed == nil {
			continue
		}
		if parsed.IsLoopback() {
			continue
		}

		addr := net.JoinHostPort(ip, "53")
		if !seen[addr] {
			seen[addr] = true
			servers = append(servers, addr)
		}
	}

	return servers
}
