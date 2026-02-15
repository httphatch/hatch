package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Hatch to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		name := color.New(color.FgCyan, color.Bold).Sprint("Hatch")
		ver := color.New(color.FgGreen).Sprint(version)
		fmt.Printf("%s %s\n", name, ver)
		fmt.Println()
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("%s Self-update is not yet available.\n", yellow("!"))
		fmt.Println("  Download the latest release from:")
		fmt.Println("  https://github.com/httphatch/hatch/releases")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
