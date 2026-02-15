package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/daemon"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the Hatch daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDown()
	},
}

func runDown() error {
	// Check if the launchd job is loaded (idempotent).
	if !daemon.IsLoaded() {
		fmt.Printf("%s is not running\n", color.CyanString("Hatch"))
		return nil
	}

	// Unload sends SIGTERM → graceful shutdown.
	if err := daemon.UnloadPlist(); err != nil {
		return fmt.Errorf("unload plist: %w", err)
	}
	log.Debug().Msg("plist unloaded")

	// Remove plist to prevent KeepAlive restart.
	if err := daemon.UninstallPlist(); err != nil {
		return fmt.Errorf("uninstall plist: %w", err)
	}
	log.Debug().Msg("plist removed")

	fmt.Printf("%s stopped\n", color.New(color.FgCyan, color.Bold).Sprint("Hatch"))
	return nil
}

func init() {
	rootCmd.AddCommand(downCmd)
}
