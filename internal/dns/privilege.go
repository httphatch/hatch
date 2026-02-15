package dns

// CommandRunner is the interface for executing shell commands, allowing
// injection for testing.
type CommandRunner interface {
	Run(command string) error
}
