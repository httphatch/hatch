package certs

import (
	"errors"
	"strings"
	"testing"
)

// MockRunner records commands instead of executing them.
type MockRunner struct {
	Commands [][]string
	Err      error // error to return from Run, if any
}

func (m *MockRunner) Run(args ...string) error {
	m.Commands = append(m.Commands, args)
	return m.Err
}

func (m *MockRunner) WriteFile(path string, content string) error {
	return m.Err
}

func TestTrustCA(t *testing.T) {
	runner := &MockRunner{}
	certPath := "/tmp/certs/rootCA.pem"

	if err := TrustCA(runner, certPath); err != nil {
		t.Fatalf("TrustCA: %v", err)
	}

	if len(runner.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runner.Commands))
	}

	want := []string{"security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", "/Library/Keychains/System.keychain", "/tmp/certs/rootCA.pem"}
	got := runner.Commands[0]
	if len(got) != len(want) {
		t.Fatalf("expected %d args, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestUntrustCA(t *testing.T) {
	runner := &MockRunner{}
	certPath := "/tmp/certs/rootCA.pem"

	if err := UntrustCA(runner, certPath); err != nil {
		t.Fatalf("UntrustCA: %v", err)
	}

	if len(runner.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runner.Commands))
	}

	want := []string{"security", "remove-trusted-cert", "-d", "/tmp/certs/rootCA.pem"}
	got := runner.Commands[0]
	if len(got) != len(want) {
		t.Fatalf("expected %d args, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTrustCA_Error(t *testing.T) {
	runner := &MockRunner{Err: errors.New("permission denied")}

	err := TrustCA(runner, "/tmp/certs/rootCA.pem")
	if err == nil {
		t.Fatal("expected error from TrustCA")
	}
	if !strings.Contains(err.Error(), "trusting CA certificate") {
		t.Errorf("unexpected error message: %v", err)
	}
}
