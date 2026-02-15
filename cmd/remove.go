package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
)

var removeCmd = &cobra.Command{
	Use:               "remove <project>",
	Aliases:           []string{"rm"},
	Short:             "Remove a project",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE:              runRemove,
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	force, _ := cmd.Flags().GetBool("force")

	cfg, err := config.LoadRaw()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if _, exists := cfg.Projects[name]; !exists {
		return fmt.Errorf("project %q not found", name)
	}

	if !force {
		fmt.Printf("Remove project '%s'? [y/N] ", name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := config.UnmergeProject(&cfg, name); err != nil {
		return fmt.Errorf("remove project: %w", err)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Project '%s' removed\n", green("✓"), name)
	return nil
}

func init() {
	removeCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
	rootCmd.AddCommand(removeCmd)
}
