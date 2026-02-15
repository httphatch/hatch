package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// sudoRunner executes commands via sudo, inheriting the terminal's TTY
// so the password prompt works.
type sudoRunner struct{}

func (r *sudoRunner) Run(command string) error {
	cmd := exec.Command("sudo", "sh", "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo command failed: %s: %w", strings.TrimSpace(command), err)
	}
	return nil
}
