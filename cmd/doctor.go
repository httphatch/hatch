package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/daemon"
	"github.com/httphatch/hatch/internal/dns"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health and diagnose common issues",
	Long:  `Runs a series of checks to verify that Hatch is correctly configured and all dependencies are in place.`,
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()

	fmt.Printf("%s\n\n", cyan("Hatch Doctor"))

	passed := 0
	total := 0

	pass := func(msg string) {
		fmt.Printf("  %s %s\n", green("✓"), msg)
		passed++
		total++
	}

	fail := func(msg, hint string) {
		fmt.Printf("  %s %s\n", red("✗"), msg)
		if hint != "" {
			fmt.Printf("    %s %s\n", yellow("→"), hint)
		}
		total++
	}

	// Check 1: Config file valid
	cfg, err := config.Load()
	if err != nil {
		var ve *config.ValidationErrors
		if errors.As(err, &ve) {
			for i, e := range ve.Errs {
				fail(fmt.Sprintf("Config error %d: %s", i+1, e), "")
			}
			fmt.Printf("    %s Fix the errors in ~/.hatch/config.yml or run 'hatch init'\n", yellow("→"))
		} else {
			fail(fmt.Sprintf("Config file is invalid: %v", err), "Fix the errors in ~/.hatch/config.yml or run 'hatch init'")
		}
		// Without valid config, skip downstream checks that need it.
		total += 8
		fmt.Println()
		fmt.Printf("%s\n", yellow(fmt.Sprintf("%d/%d checks passed", passed, total)))
		return nil
	}
	pass("Config file is valid")

	// Check 2: DNS resolver installed
	tld := cfg.Settings.TLD
	if dns.IsResolverInstalled(tld) {
		pass(fmt.Sprintf("DNS resolver installed (.%s)", tld))
	} else {
		fail(fmt.Sprintf("DNS resolver not installed (.%s)", tld), "Run 'hatch init' to install the DNS resolver")
	}

	// Check 3: Root CA exists
	caPaths := certs.NewCAPaths(config.CertsDir())
	if certs.CAExists(caPaths) {
		pass("Root CA exists")
	} else {
		fail("Root CA not found", "Run 'hatch init' to generate a root CA")
	}

	// Check 4: Root CA trusted in Keychain
	if certs.IsCATrusted(caPaths.Cert) {
		pass("Root CA trusted in Keychain")
	} else {
		fail("Root CA is not trusted", "Run 'hatch trust' to re-trust the root CA")
	}

	// Check 5: Intermediate CA exists
	if certs.IntermediateCAExists(caPaths) {
		pass("Intermediate CA exists")
	} else {
		fail("Intermediate CA not found", "Run 'hatch up' to generate an intermediate CA")
	}

	// Check 6: Launchd plist installed
	plistPath, err := daemon.PlistPath()
	if err != nil {
		fail(fmt.Sprintf("Could not determine plist path: %v", err), "")
	} else if _, err := os.Stat(plistPath); err == nil {
		pass("Launchd plist installed")
	} else {
		fail("Launchd plist not installed", "Run 'hatch up' to install and start the daemon")
	}

	// Check 6: Launchd plist loaded
	if daemon.IsLoaded() {
		pass("Launchd plist loaded")
	} else {
		fail("Launchd plist not loaded", "Run 'hatch up' to start the daemon")
	}

	// Check 7: Ports available
	running, _, _ := daemon.IsRunning()
	httpPort := cfg.Settings.HTTPPort
	httpsPort := cfg.Settings.HTTPSPort

	if running {
		// Daemon is running — verify it's actually listening on the ports.
		if checkDaemonListening(httpPort) {
			pass(fmt.Sprintf("HTTP port :%d is reachable", httpPort))
		} else {
			fail(fmt.Sprintf("HTTP port :%d is not reachable", httpPort), "The daemon is running but not listening on this port")
		}
		if checkDaemonListening(httpsPort) {
			pass(fmt.Sprintf("HTTPS port :%d is reachable", httpsPort))
		} else {
			fail(fmt.Sprintf("HTTPS port :%d is not reachable", httpsPort), "The daemon is running but not listening on this port")
		}
	} else {
		// Daemon is not running — check that ports are free.
		if checkPortAvailable(httpPort) {
			pass(fmt.Sprintf("HTTP port :%d is available", httpPort))
		} else {
			hint := portConflictHint(httpPort)
			fail(fmt.Sprintf("HTTP port :%d is in use", httpPort), hint)
		}
		if checkPortAvailable(httpsPort) {
			pass(fmt.Sprintf("HTTPS port :%d is available", httpsPort))
		} else {
			hint := portConflictHint(httpsPort)
			fail(fmt.Sprintf("HTTPS port :%d is in use", httpsPort), hint)
		}
	}

	// Check 8: No stale projects
	staleProjects := findStaleProjects(cfg.Projects)
	if len(staleProjects) == 0 {
		pass("No stale projects")
	} else {
		for _, name := range staleProjects {
			fail(fmt.Sprintf("Stale project: %s (path not found)", name), fmt.Sprintf("Run 'hatch remove %s' to clean up stale entries", name))
		}
	}

	// Summary
	fmt.Println()
	if passed == total {
		fmt.Printf("%s\n", green(fmt.Sprintf("All %d checks passed", total)))
	} else {
		fmt.Printf("%s\n", yellow(fmt.Sprintf("%d/%d checks passed", passed, total)))
	}

	return nil
}

func checkPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func checkDaemonListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func portConflictHint(port int) string {
	info, err := daemon.CheckPort(port)
	if err != nil || info == nil {
		return "Another process is using this port"
	}
	return fmt.Sprintf("In use by %s", info)
}

func findStaleProjects(projects map[string]config.Project) []string {
	var stale []string
	for name, proj := range projects {
		if _, err := os.Stat(proj.Path); os.IsNotExist(err) {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	return stale
}
