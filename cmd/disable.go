package cmd

import (
	"github.com/spf13/cobra"
)

var disableCmd = &cobra.Command{
	Use:               "disable <project>",
	Short:             "Disable a project",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		return setProjectEnabled(args[0], false)
	},
}

func init() {
	rootCmd.AddCommand(disableCmd)
}
