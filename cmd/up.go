package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/api"
	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/daemon"
	"github.com/httphatch/hatch/internal/dns"
)

var upCmd = &cobra.Command{
	Use:     "up",
	Aliases: []string{"start"},
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

	// Pre-flight: catch port conflicts before loading the plist so the error
	// is visible in the terminal. The daemon performs the same check at startup.
	for _, port := range []int{cfg.Settings.HTTPPort, cfg.Settings.HTTPSPort} {
		info, err := daemon.CheckPort(port)
		if err != nil {
			log.Warn().Err(err).Int("port", port).Msg("could not check port availability")
		} else if info != nil {
			return fmt.Errorf("port %d is already in use by %s; stop that process first", port, info)
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

	// Verify the daemon actually started by polling the API.
	if err := waitForDaemon(); err != nil {
		return fmt.Errorf("daemon failed to start; check logs with: hatch logs")
	}

	// Launch the tray app as a separate process if enabled.
	if cfg.Settings.TrayIcon {
		launchTray(false)
	}

	fmt.Printf("%s started\n", color.New(color.FgCyan, color.Bold).Sprint("Hatch"))
	return nil
}

// waitForDaemon polls the daemon API until it responds or the timeout expires.
func waitForDaemon() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := &http.Client{}
	url := "http://" + api.DefaultAddr + "/api/status"

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for daemon")
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func init() {
	rootCmd.AddCommand(upCmd)
}
