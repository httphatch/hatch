package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/api"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/daemon"
	"github.com/httphatch/hatch/internal/logging"
)

var runCmd = &cobra.Command{
	Use:    "_run",
	Short:  "Run the daemon (internal, used by launchd)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, syscall.SIGINT)
		defer stop()

		// Best-effort config load to read settings; use defaults on failure.
		defaults := config.DefaultConfig()
		logLevel := defaults.Settings.LogLevel
		trayIcon := defaults.Settings.TrayIcon
		if cfg, err := config.LoadWithProjectConfigs(); err == nil {
			logLevel = cfg.Settings.LogLevel
			trayIcon = cfg.Settings.TrayIcon
		}

		logHub := api.NewLogHub()

		w, err := logging.Setup(logging.Config{
			FilePath:    config.LogFile(),
			Level:       logLevel,
			ExtraWriter: logHub,
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to setup logging")
			os.Exit(1)
		}
		defer func() { _ = w.Close() }()

		d := daemon.New(version, logHub)
		if err := runDaemon(ctx, d, trayIcon); err != nil {
			log.Error().Err(err).Msg("daemon exited with error")
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
