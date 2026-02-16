package cmd

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/daemon"
	"github.com/httphatch/hatch/internal/update"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon state, projects, and service health",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// Start update check in background so it doesn't block output.
	var updateHint string
	var updateWg sync.WaitGroup
	updateWg.Add(1)
	go func() {
		defer updateWg.Done()
		updateHint = checkUpdateHint()
	}()

	// Daemon state
	running, pid, _ := daemon.IsRunning()
	if running {
		if pid > 0 {
			fmt.Printf("Daemon: %s (pid %d)\n", green("running"), pid)
		} else {
			fmt.Printf("Daemon: %s\n", green("running"))
		}
	} else {
		fmt.Printf("Daemon: %s\n", red("not running"))
		fmt.Printf("  %s Run 'hatch up' to start the daemon\n", yellow("→"))
	}

	if len(cfg.Projects) == 0 {
		fmt.Println()
		fmt.Println("No projects configured.")
		return nil
	}

	// Sort project names
	names := make([]string, 0, len(cfg.Projects))
	for name := range cfg.Projects {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		proj := cfg.Projects[name]

		fmt.Println()

		// Project header
		if proj.Enabled {
			fmt.Printf("%s (%s) %s enabled\n", name, proj.Domain, green("✓"))
		} else {
			fmt.Printf("%s (%s) %s disabled\n", name, proj.Domain, red("✗"))
		}

		if !proj.Enabled {
			fmt.Println("  (skipped — project disabled)")
			continue
		}

		// Sort service names
		svcNames := make([]string, 0, len(proj.Services))
		for svcName := range proj.Services {
			svcNames = append(svcNames, svcName)
		}
		sort.Strings(svcNames)

		// Compute column widths
		nameW, domainW, upstreamW := len("SERVICE"), len("DOMAIN"), len("UPSTREAM")
		type row struct {
			name, domain, upstream, status string
		}
		rows := make([]row, 0, len(svcNames))

		for _, svcName := range svcNames {
			svc := proj.Services[svcName]
			domain := resolveDomain(proj, svc)
			upstream := extractDialAddr(svc.Proxy)

			var status string
			if upstream != "" && dialHealth(upstream) {
				status = green("✓") + " healthy"
			} else {
				status = red("✗") + " unhealthy"
			}

			if len(svcName) > nameW {
				nameW = len(svcName)
			}
			if len(domain) > domainW {
				domainW = len(domain)
			}
			if len(upstream) > upstreamW {
				upstreamW = len(upstream)
			}

			rows = append(rows, row{svcName, domain, upstream, status})
		}

		// Print table header
		fmt.Printf("  %-*s  %-*s  %-*s  %s\n", nameW, "SERVICE", domainW, "DOMAIN", upstreamW, "UPSTREAM", "STATUS")

		// Print rows
		for _, r := range rows {
			fmt.Printf("  %-*s  %-*s  %-*s  %s\n", nameW, r.name, domainW, r.domain, upstreamW, r.upstream, r.status)
		}
	}

	updateWg.Wait()
	if updateHint != "" {
		fmt.Print(updateHint)
	}

	return nil
}

// dialHealth performs a TCP dial to check if a service is reachable.
func dialHealth(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// extractDialAddr parses a proxy URL and returns the host:port for dialing.
func extractDialAddr(proxyURL string) string {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "https":
			port = "443"
		default:
			port = "80"
		}
	}
	return net.JoinHostPort(host, port)
}

// resolveDomain returns the full domain for a service, prepending the
// subdomain if configured.
func resolveDomain(proj config.Project, svc config.Service) string {
	if svc.Subdomain != "" {
		return svc.Subdomain + "." + proj.Domain
	}
	return proj.Domain
}

// checkUpdateHint silently checks for a newer version and returns a
// formatted hint string, or empty if up to date / check fails.
func checkUpdateHint() string {
	currentRaw := strings.TrimPrefix(version, "v")
	current, err := semver.NewVersion(currentRaw)
	if err != nil {
		return ""
	}

	client := &http.Client{Timeout: 3 * time.Second}
	rel, err := update.CheckLatest(client)
	if err != nil {
		return ""
	}

	latestRaw := strings.TrimPrefix(rel.TagName, "v")
	latest, err := semver.NewVersion(latestRaw)
	if err != nil {
		return ""
	}

	if current.LessThan(latest) {
		yellow := color.New(color.FgYellow).SprintFunc()
		return fmt.Sprintf("\n%s Update available: v%s → v%s (run 'hatch update')\n",
			yellow("!"), current, latest)
	}
	return ""
}
