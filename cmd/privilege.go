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

func (r *sudoRunner) Run(args ...string) error {
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo command failed: %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func (r *sudoRunner) WriteFile(path string, content string) error {
	cmd := exec.Command("sudo", "tee", path)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = nil // suppress tee's stdout echo
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo write %s failed: %w", path, err)
	}
	return nil
}
