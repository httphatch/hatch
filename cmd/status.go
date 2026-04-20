package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
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
	cfg, err := config.LoadWithProjectConfigs()
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

	// Fetch tunnel statuses from daemon in background.
	type tunnelInfo struct {
		Project string `json:"project"`
		Service string `json:"service"`
		URL     string `json:"url"`
		Type    string `json:"type"`
		Running bool   `json:"running"`
	}
	tunnelMap := make(map[string]tunnelInfo) // "project/service" -> info
	var tunnelWg sync.WaitGroup
	tunnelWg.Add(1)
	go func() {
		defer tunnelWg.Done()
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(daemonBaseURL() + "/api/tunnels")
		if err != nil {
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return
		}
		var tunnels []tunnelInfo
		if err := json.NewDecoder(resp.Body).Decode(&tunnels); err != nil {
			return
		}
		for _, t := range tunnels {
			if t.Running {
				tunnelMap[t.Project+"/"+t.Service] = t
			}
		}
	}()

	// Port check cache and conflict collector.
	portCache := make(map[int]*daemon.PortInfo)
	var conflicts []portConflict

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
		for _, dp := range []int{cfg.Settings.HTTPPort, cfg.Settings.HTTPSPort} {
			_, c := diagnosePort(dp, portCache)
			if c != nil {
				if c.pid > 0 {
					fmt.Printf("  %s Port %d in use by %s (PID %d)\n", red("!"), c.port, c.process, c.pid)
				} else {
					fmt.Printf("  %s Port %d in use by %s\n", red("!"), c.port, c.process)
				}
				conflicts = append(conflicts, *c)
			}
		}
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

		// Wait for tunnel data before rendering the table.
		tunnelWg.Wait()

		// Compute column widths
		nameW, domainW, upstreamW, tunnelW := len("SERVICE"), len("DOMAIN"), len("UPSTREAM"), 0
		type row struct {
			name, domain, upstream, status, tunnelURL string
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
				suffix := ""
				if upstream != "" {
					port := extractPort(upstream)
					var c *portConflict
					suffix, c = diagnosePort(port, portCache)
					if c != nil {
						conflicts = append(conflicts, *c)
					}
				}
				status = red("✗") + " unhealthy" + suffix
			}

			tunnelURL := ""
			if t, ok := tunnelMap[name+"/"+svcName]; ok {
				tunnelURL = t.URL
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
			if len(tunnelURL) > tunnelW {
				tunnelW = len(tunnelURL)
			}

			rows = append(rows, row{svcName, domain, upstream, status, tunnelURL})
		}

		// Check if any tunnels are active for this project.
		hasTunnels := false
		for _, r := range rows {
			if r.tunnelURL != "" {
				hasTunnels = true
				break
			}
		}

		// Print table header
		if hasTunnels {
			if tunnelW < len("TUNNEL") {
				tunnelW = len("TUNNEL")
			}
			fmt.Printf("  %-*s  %-*s  %-*s  %-*s  %s\n", nameW, "SERVICE", domainW, "DOMAIN", upstreamW, "UPSTREAM", tunnelW, "TUNNEL", "STATUS")
		} else {
			fmt.Printf("  %-*s  %-*s  %-*s  %s\n", nameW, "SERVICE", domainW, "DOMAIN", upstreamW, "UPSTREAM", "STATUS")
		}

		// Print rows
		for _, r := range rows {
			if hasTunnels {
				fmt.Printf("  %-*s  %-*s  %-*s  %-*s  %s\n", nameW, r.name, domainW, r.domain, upstreamW, r.upstream, tunnelW, r.tunnelURL, r.status)
			} else {
				fmt.Printf("  %-*s  %-*s  %-*s  %s\n", nameW, r.name, domainW, r.domain, upstreamW, r.upstream, r.status)
			}
		}
	}

	// Deduplicate and print conflict summary.
	if len(conflicts) > 0 {
		seen := make(map[int]bool)
		var unique []portConflict
		for _, c := range conflicts {
			if !seen[c.port] {
				seen[c.port] = true
				unique = append(unique, c)
			}
		}
		fmt.Println()
		fmt.Println("Port conflicts detected:")
		for _, c := range unique {
			if c.pid > 0 {
				fmt.Printf("  Port %d in use by %s (PID %d)  %s kill %d\n", c.port, c.process, c.pid, yellow("→"), c.pid)
			} else {
				fmt.Printf("  Port %d in use by %s\n", c.port, c.process)
			}
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

// extractPort returns the numeric port from a "host:port" string, or 0 if unparsable.
func extractPort(addr string) int {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	p, err := strconv.Atoi(portStr)
	if err != nil {
		return 0
	}
	return p
}

// portConflict records a port held by another process.
type portConflict struct {
	port    int
	process string
	pid     int
}

// diagnosePort checks who holds a port and returns a status suffix and optional conflict.
// Results are cached in the provided map to avoid duplicate lsof calls.
func diagnosePort(port int, cache map[int]*daemon.PortInfo) (string, *portConflict) {
	if port == 0 {
		return "", nil
	}
	info, cached := cache[port]
	if !cached {
		var err error
		info, err = daemon.CheckPort(port)
		if err != nil {
			return "", nil
		}
		cache[port] = info
	}
	if info == nil {
		return fmt.Sprintf(" (not listening on :%d)", port), nil
	}
	conflict := &portConflict{port: port, process: info.Process, pid: info.PID}
	return fmt.Sprintf(" (port :%d in use by %s)", port, info), conflict
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
