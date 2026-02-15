package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the Hatch daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runDown(); err != nil {
			return err
		}
		if err := runUp(); err != nil {
			return err
		}
		fmt.Printf("%s restarted\n", color.New(color.FgCyan, color.Bold).Sprint("Hatch"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
