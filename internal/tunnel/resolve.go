package tunnel

import (
	"github.com/rs/zerolog/log"

	"github.com/httphatch/hatch/internal/config"
)

// HostnameResolver fetches the hostnames attached to a named Cloudflare tunnel.
type HostnameResolver interface {
	GetTunnelHostnames(apiToken, accountID, tunnelName string) ([]string, error)
}

// ResolveTunnelDomains queries the Cloudflare API for each named tunnel in the
// config and returns a map of "project/service" to the hostnames configured in
// the tunnel's ingress rules. Failures are logged as warnings; partial results
// are returned so the daemon is never blocked by a Cloudflare API issue.
func ResolveTunnelDomains(cfg config.Config, resolver HostnameResolver) map[string][]string {
	result := make(map[string][]string)

	for projName, proj := range cfg.Projects {
		if !proj.Enabled {
			continue
		}
		for svcName, svc := range proj.Services {
			tunnelVal := string(svc.Tunnel)
			if tunnelVal == "" || tunnelVal == "true" {
				continue
			}

			token := proj.CloudflareToken
			if token == "" {
				token = cfg.Settings.CloudflareToken
			}
			if token == "" {
				continue
			}

			accountID := cfg.Settings.CloudflareAccountID

			hostnames, err := resolver.GetTunnelHostnames(token, accountID, tunnelVal)
			if err != nil {
				log.Warn().
					Err(err).
					Str("project", projName).
					Str("service", svcName).
					Str("tunnel", tunnelVal).
					Msg("failed to resolve tunnel hostnames")
				continue
			}

			if len(hostnames) > 0 {
				key := projName + "/" + svcName
				result[key] = hostnames
			}
		}
	}

	return result
}
