package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Open the Hatch dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		if runtime.GOOS != "darwin" {
			return fmt.Errorf("dashboard is only supported on macOS")
		}
		launchTray(true)
		fmt.Println("Opening dashboard...")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(appCmd)
}
