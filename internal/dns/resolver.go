package dns

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
)

// ResolverFileContent returns the content for a macOS resolver file
// that directs queries to the given IP and port.
func ResolverFileContent(listenIP string, port int) string {
	return fmt.Sprintf("nameserver %s\nport %d\n", listenIP, port)
}

// ResolverFilePath returns the full path for a resolver file for the
// given TLD (e.g. "/etc/resolver/test").
func ResolverFilePath(tld string) string {
	return filepath.Join(ResolverDir, tld)
}

// safeTLD matches only simple alphanumeric TLD values.
var safeTLD = regexp.MustCompile(`^[a-z]+$`)

// InstallResolverFile creates the resolver directory and writes the
// resolver file for the given TLD. The operation requires elevated
// privileges, so it uses the provided CommandRunner.
func InstallResolverFile(runner CommandRunner, tld, listenIP string, port int) error {
	if err := validateResolverInputs(tld, listenIP, port); err != nil {
		return err
	}
	content := ResolverFileContent(listenIP, port)
	path := ResolverFilePath(tld)
	cmd := fmt.Sprintf("mkdir -p %s && printf '%%s' '%s' > %s", ResolverDir, content, path)
	if err := runner.Run(cmd); err != nil {
		return fmt.Errorf("installing resolver file for %s: %w", tld, err)
	}
	return nil
}

// RemoveResolverFile removes the resolver file for the given TLD.
// The operation requires elevated privileges, so it uses the provided
// CommandRunner.
func RemoveResolverFile(runner CommandRunner, tld string) error {
	if !safeTLD.MatchString(tld) {
		return fmt.Errorf("invalid TLD %q", tld)
	}
	path := ResolverFilePath(tld)
	cmd := fmt.Sprintf("rm -f %s", path)
	if err := runner.Run(cmd); err != nil {
		return fmt.Errorf("removing resolver file for %s: %w", tld, err)
	}
	return nil
}

func validateResolverInputs(tld, listenIP string, port int) error {
	if !safeTLD.MatchString(tld) {
		return fmt.Errorf("invalid TLD %q", tld)
	}
	if net.ParseIP(listenIP) == nil {
		return fmt.Errorf("invalid listen IP %q", listenIP)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %d", port)
	}
	return nil
}

// IsResolverInstalled reports whether a resolver file exists for the
// given TLD.
func IsResolverInstalled(tld string) bool {
	_, err := os.Stat(ResolverFilePath(tld))
	return err == nil
}
