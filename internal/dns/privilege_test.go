package dns

import "testing"

// MockRunner records commands instead of executing them.
type MockRunner struct {
	Commands []string
	Err      error // error to return from Run, if any
}

func (m *MockRunner) Run(command string) error {
	m.Commands = append(m.Commands, command)
	return m.Err
}

func TestMockRunner_RecordsCommands(t *testing.T) {
	runner := &MockRunner{}

	_ = runner.Run("echo hello")
	_ = runner.Run("echo world")

	if len(runner.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(runner.Commands))
	}
	if runner.Commands[0] != "echo hello" {
		t.Errorf("expected 'echo hello', got %q", runner.Commands[0])
	}
	if runner.Commands[1] != "echo world" {
		t.Errorf("expected 'echo world', got %q", runner.Commands[1])
	}
}
