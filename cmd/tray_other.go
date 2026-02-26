//go:build !darwin

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var trayCmd = &cobra.Command{
	Use:    "_tray",
	Short:  "Run the tray app (internal)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("tray is not supported on this platform")
	},
}

func init() {
	rootCmd.AddCommand(trayCmd)
}
