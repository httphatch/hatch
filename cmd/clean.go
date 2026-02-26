package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/caddy"
	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/daemon"
	"github.com/httphatch/hatch/internal/dns"
)

var cleanCmd = &cobra.Command{
	Use:     "clean",
	Aliases: []string{"uninstall"},
	Short:   "Remove all Hatch configuration and system integrations",
	Long:    `Stops the daemon, removes the DNS resolver, untrusts the root CA, and deletes the ~/.hatch directory.`,
	RunE:    runClean,
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) error {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()

	fmt.Printf("Cleaning up %s...\n\n", cyan("Hatch"))

	// Load config before deleting it — we need the TLD for DNS cleanup.
	// Fall back to default TLD if config can't be loaded.
	tld := "test"
	if cfg, err := config.Load(); err == nil {
		tld = cfg.Settings.TLD
	}

	// Resolve CA cert path before deleting the config directory.
	caPaths := certs.NewCAPaths(config.CertsDir())

	anyCleaned := false

	// Step 1: Stop daemon
	if daemon.IsLoaded() {
		if err := daemon.UninstallPlist(); err != nil {
			fmt.Printf("  %s Failed to remove daemon plist: %v\n", red("✗"), err)
			os.Exit(1)
		}
		if err := daemon.RemoveJob(); err != nil {
			fmt.Printf("  %s Failed to stop daemon: %v\n", red("✗"), err)
			os.Exit(1)
		}
		fmt.Printf("  %s Daemon stopped\n", green("✓"))
		anyCleaned = true
	} else {
		// Still remove the plist file if it exists even when not loaded.
		_ = daemon.UninstallPlist()
		fmt.Printf("  %s Daemon not running\n", green("✓"))
	}

	// Step 2: Remove DNS resolver
	if dns.IsResolverInstalled(tld) {
		if err := dns.RemoveResolverFile(&sudoRunner{}, tld); err != nil {
			fmt.Printf("  %s Failed to remove DNS resolver: %v\n", red("✗"), err)
			os.Exit(1)
		}
		fmt.Printf("  %s DNS resolver removed\n", green("✓"))
		anyCleaned = true
	} else {
		fmt.Printf("  %s DNS resolver not installed\n", green("✓"))
	}

	// Step 3: Untrust CA from Keychain
	if certs.CAExists(caPaths) && certs.IsCATrusted(caPaths.Cert) {
		if err := certs.UntrustCA(&sudoRunner{}, caPaths.Cert); err != nil {
			fmt.Printf("  %s Failed to untrust root CA: %v\n", red("✗"), err)
			os.Exit(1)
		}
		fmt.Printf("  %s Root CA untrusted\n", green("✓"))
		anyCleaned = true
	} else {
		fmt.Printf("  %s Root CA not trusted\n", green("✓"))
	}

	// Step 4: Delete Caddy data directory
	caddyDataDir := caddy.DataDir()
	if dirExistsAt(caddyDataDir) {
		if err := os.RemoveAll(caddyDataDir); err != nil {
			fmt.Printf("  %s Failed to remove Caddy data directory: %v\n", red("✗"), err)
			os.Exit(1)
		}
		fmt.Printf("  %s Caddy data directory removed\n", green("✓"))
		anyCleaned = true
	} else {
		fmt.Printf("  %s Caddy data directory not found\n", green("✓"))
	}

	// Step 5: Delete ~/.hatch directory
	if dirExistsAt(config.Dir()) {
		if err := os.RemoveAll(config.Dir()); err != nil {
			fmt.Printf("  %s Failed to remove config directory: %v\n", red("✗"), err)
			os.Exit(1)
		}
		fmt.Printf("  %s Config directory removed (~/.hatch)\n", green("✓"))
		anyCleaned = true
	} else {
		fmt.Printf("  %s Config directory not found\n", green("✓"))
	}

	fmt.Println()
	if anyCleaned {
		fmt.Printf("%s has been cleaned up.\n", cyan("Hatch"))
	} else {
		fmt.Println("Nothing to clean up.")
	}

	return nil
}
