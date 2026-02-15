package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for Hatch.

To load completions:

Bash:
  $ source <(hatch completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ hatch completion bash > /etc/bash_completion.d/hatch
  # macOS:
  $ hatch completion bash > $(brew --prefix)/etc/bash_completion.d/hatch

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ hatch completion zsh > "${fpath[1]}/_hatch"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ hatch completion fish | source

  # To load completions for each session, execute once:
  $ hatch completion fish > ~/.config/fish/completions/hatch.fish
`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE:      runCompletion,
}

func runCompletion(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		return rootCmd.GenBashCompletionV2(os.Stdout, true)
	case "zsh":
		return rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		return rootCmd.GenFishCompletion(os.Stdout, true)
	default:
		return fmt.Errorf("unsupported shell %q — use bash, zsh, or fish", args[0])
	}
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
