package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the Hatch daemon logs",
	RunE:  runLogs,
}

func runLogs(cmd *cobra.Command, args []string) error {
	path := config.LogFile()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("log file not found at %s — is the daemon running?", path)
	}

	follow, _ := cmd.Flags().GetBool("follow")
	lines, _ := cmd.Flags().GetInt("lines")

	if follow {
		tailCmd := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), "-F", path)
		tailCmd.Stdin = os.Stdin
		tailCmd.Stdout = os.Stdout
		tailCmd.Stderr = os.Stderr
		return tailCmd.Run()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read log file: %w", err)
	}

	content := string(data)
	if content == "" {
		fmt.Println("Log file is empty.")
		return nil
	}

	// Print last N lines.
	allLines := splitLines(content)
	start := len(allLines) - lines
	if start < 0 {
		start = 0
	}
	for _, line := range allLines[start:] {
		fmt.Println(line)
	}

	return nil
}

// splitLines splits text into lines, handling trailing newline gracefully.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func init() {
	logsCmd.Flags().BoolP("follow", "f", false, "follow log output")
	logsCmd.Flags().IntP("lines", "n", 50, "number of lines to show")
	rootCmd.AddCommand(logsCmd)
}
