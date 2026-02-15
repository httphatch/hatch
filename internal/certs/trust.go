package certs

import (
	"fmt"
	"os/exec"
)

// CommandRunner is the interface for executing shell commands, allowing
// injection for testing.
type CommandRunner interface {
	Run(command string) error
}

// TrustCA adds the CA certificate to the macOS System Keychain as a
// trusted root. Requires elevated privileges via the provided runner.
func TrustCA(runner CommandRunner, certPath string) error {
	cmd := fmt.Sprintf("security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s", certPath)
	if err := runner.Run(cmd); err != nil {
		return fmt.Errorf("trusting CA certificate: %w", err)
	}
	return nil
}

// UntrustCA removes the CA certificate from the macOS trusted certificates.
// Requires elevated privileges via the provided runner.
func UntrustCA(runner CommandRunner, certPath string) error {
	cmd := fmt.Sprintf("security remove-trusted-cert -d %s", certPath)
	if err := runner.Run(cmd); err != nil {
		return fmt.Errorf("untrusting CA certificate: %w", err)
	}
	return nil
}

// IsCATrusted reports whether the certificate at certPath is trusted by
// the system. It runs the non-privileged security verify-cert command.
func IsCATrusted(certPath string) bool {
	err := exec.Command("security", "verify-cert", "-c", certPath).Run()
	return err == nil
}
