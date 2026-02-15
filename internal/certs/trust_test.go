package certs

import (
	"errors"
	"strings"
	"testing"
)

// MockRunner records commands instead of executing them.
type MockRunner struct {
	Commands []string
	Err      error // error to return from Run, if any
}

func (m *MockRunner) Run(command string) error {
	m.Commands = append(m.Commands, command)
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

	want := "security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain /tmp/certs/rootCA.pem"
	if runner.Commands[0] != want {
		t.Errorf("unexpected command:\n  got:  %s\n  want: %s", runner.Commands[0], want)
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

	want := "security remove-trusted-cert -d /tmp/certs/rootCA.pem"
	if runner.Commands[0] != want {
		t.Errorf("unexpected command:\n  got:  %s\n  want: %s", runner.Commands[0], want)
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

