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

	// Remove plist first so KeepAlive cannot respawn the process.
	if err := daemon.UninstallPlist(); err != nil {
		return fmt.Errorf("uninstall plist: %w", err)
	}
	log.Debug().Msg("plist removed")

	// Remove the job by label. This works after the plist file is deleted,
	// unlike "launchctl unload" which requires the file to exist.
	if err := daemon.RemoveJob(); err != nil {
		return fmt.Errorf("remove job: %w", err)
	}
	log.Debug().Msg("job removed")

	fmt.Printf("%s stopped\n", color.New(color.FgCyan, color.Bold).Sprint("Hatch"))
	return nil
}

func init() {
	rootCmd.AddCommand(downCmd)
}
