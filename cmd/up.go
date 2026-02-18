package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/daemon"
	"github.com/httphatch/hatch/internal/dns"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the Hatch daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUp()
	},
}

func runUp() error {
	// Check if already running (idempotent).
	running, pid, err := daemon.IsRunning()
	if err != nil {
		return fmt.Errorf("check running: %w", err)
	}
	if running {
		fmt.Printf("%s already running (pid %d)\n", color.CyanString("Hatch"), pid)
		return nil
	}

	// Initialize config directory and default config file.
	if err := config.Init(); err != nil {
		return fmt.Errorf("init config: %w", err)
	}
	log.Debug().Msg("config initialized")

	// Load config to read settings.
	cfg, err := config.LoadWithProjectConfigs()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Generate CA if needed.
	caPaths := certs.NewCAPaths(config.CertsDir())
	if !certs.CAExists(caPaths) {
		log.Info().Msg("generating root CA")
		if err := certs.GenerateCA(caPaths); err != nil {
			return fmt.Errorf("generate CA: %w", err)
		}
	}

	// Generate intermediate CA if needed.
	if !certs.IntermediateCAExists(caPaths) {
		log.Info().Msg("generating intermediate CA")
		if err := certs.GenerateIntermediateCA(caPaths); err != nil {
			return fmt.Errorf("generate intermediate CA: %w", err)
		}
	}

	// Trust CA if needed.
	if !certs.IsCATrusted(caPaths.Cert) {
		log.Info().Msg("trusting root CA (may prompt for password)")
		if err := certs.TrustCA(&sudoRunner{}, caPaths.Cert); err != nil {
			return fmt.Errorf("trust CA: %w", err)
		}
	}

	// Install DNS resolver if needed.
	if !dns.IsResolverInstalled(cfg.Settings.TLD) {
		log.Info().Str("tld", cfg.Settings.TLD).Msg("installing DNS resolver (may prompt for password)")
		if err := dns.InstallResolverFile(&sudoRunner{}, cfg.Settings.TLD, dns.DefaultListenIP, dns.DefaultPort); err != nil {
			return fmt.Errorf("install resolver: %w", err)
		}
	}

	// Build launchd config and install plist.
	launchdCfg, err := daemon.DefaultLaunchdConfig(cfg.Settings.AutoStart)
	if err != nil {
		return fmt.Errorf("launchd config: %w", err)
	}

	if err := daemon.InstallPlist(launchdCfg); err != nil {
		return fmt.Errorf("install plist: %w", err)
	}
	log.Debug().Msg("plist installed")

	// Load the plist into launchd.
	if err := daemon.LoadPlist(); err != nil {
		return fmt.Errorf("load plist: %w", err)
	}

	fmt.Printf("%s started\n", color.New(color.FgCyan, color.Bold).Sprint("Hatch"))
	return nil
}

func init() {
	rootCmd.AddCommand(upCmd)
}
