package dns

import "testing"

// MockRunner records commands instead of executing them.
type MockRunner struct {
	Commands   [][]string
	Files      map[string]string // path -> content
	Err        error             // error to return, if any
	WriteErr   error             // error to return from WriteFile, if any
}

func (m *MockRunner) Run(args ...string) error {
	m.Commands = append(m.Commands, args)
	return m.Err
}

func (m *MockRunner) WriteFile(path string, content string) error {
	if m.Files == nil {
		m.Files = make(map[string]string)
	}
	m.Files[path] = content
	return m.WriteErr
}

func TestMockRunner_RecordsCommands(t *testing.T) {
	runner := &MockRunner{}

	_ = runner.Run("echo", "hello")
	_ = runner.Run("echo", "world")

	if len(runner.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(runner.Commands))
	}
	if runner.Commands[0][0] != "echo" || runner.Commands[0][1] != "hello" {
		t.Errorf("expected [echo hello], got %v", runner.Commands[0])
	}
	if runner.Commands[1][0] != "echo" || runner.Commands[1][1] != "world" {
		t.Errorf("expected [echo world], got %v", runner.Commands[1])
	}
}
