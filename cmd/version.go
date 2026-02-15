package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of Hatch",
	Run: func(cmd *cobra.Command, args []string) {
		name := color.New(color.FgCyan, color.Bold).Sprint("Hatch")
		ver := color.New(color.FgGreen).Sprint(version)
		fmt.Printf("%s %s\n", name, ver)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
