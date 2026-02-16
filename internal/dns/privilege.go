package dns

// CommandRunner is the interface for executing privileged commands.
// Arguments are passed individually to avoid shell interpretation.
type CommandRunner interface {
	Run(args ...string) error
	WriteFile(path string, content string) error
}
