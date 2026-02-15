package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "hatch",
	Short: "Local HTTPS reverse proxy for macOS development",
	Long:  `Hatch is a local HTTPS reverse proxy that makes developing with custom domains and TLS effortless on macOS.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level := zerolog.InfoLevel
		if verbose {
			level = zerolog.DebugLevel
		}
		log.Logger = zerolog.New(
			zerolog.ConsoleWriter{Out: os.Stderr},
		).Level(level).With().Timestamp().Logger()

		if verbose {
			log.Debug().Msg("debug logging enabled")
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
}
