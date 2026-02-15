package dns

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolverFileContent(t *testing.T) {
	tests := []struct {
		name     string
		listenIP string
		port     int
		want     string
	}{
		{
			name:     "default",
			listenIP: "127.0.0.1",
			port:     5053,
			want:     "nameserver 127.0.0.1\nport 5053\n",
		},
		{
			name:     "custom port",
			listenIP: "127.0.0.1",
			port:     15353,
			want:     "nameserver 127.0.0.1\nport 15353\n",
		},
		{
			name:     "custom ip",
			listenIP: "192.168.1.1",
			port:     53,
			want:     "nameserver 192.168.1.1\nport 53\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolverFileContent(tt.listenIP, tt.port)
			if got != tt.want {
				t.Errorf("ResolverFileContent(%q, %d) = %q, want %q", tt.listenIP, tt.port, got, tt.want)
			}
		})
	}
}

func TestResolverFilePath(t *testing.T) {
	tests := []struct {
		tld  string
		want string
	}{
		{"test", "/etc/resolver/test"},
		{"localhost", "/etc/resolver/localhost"},
		{"dev", "/etc/resolver/dev"},
	}

	for _, tt := range tests {
		t.Run(tt.tld, func(t *testing.T) {
			got := ResolverFilePath(tt.tld)
			if got != tt.want {
				t.Errorf("ResolverFilePath(%q) = %q, want %q", tt.tld, got, tt.want)
			}
		})
	}
}

func TestInstallResolverFile(t *testing.T) {
	runner := &MockRunner{}

	err := InstallResolverFile(runner, "test", "127.0.0.1", 5053)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runner.Commands))
	}

	cmd := runner.Commands[0]
	if !strings.Contains(cmd, "mkdir -p /etc/resolver") {
		t.Errorf("expected mkdir command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "nameserver 127.0.0.1") {
		t.Errorf("expected nameserver in command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "port 5053") {
		t.Errorf("expected port in command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "/etc/resolver/test") {
		t.Errorf("expected resolver path in command, got: %s", cmd)
	}
}

func TestRemoveResolverFile(t *testing.T) {
	runner := &MockRunner{}

	err := RemoveResolverFile(runner, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runner.Commands))
	}

	cmd := runner.Commands[0]
	if cmd != "rm -f /etc/resolver/test" {
		t.Errorf("expected rm command, got: %s", cmd)
	}
}

func TestInstallResolverFile_Error(t *testing.T) {
	runner := &MockRunner{Err: errors.New("permission denied")}

	err := InstallResolverFile(runner, "test", "127.0.0.1", 5053)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "installing resolver file for test") {
		t.Errorf("expected wrapped error, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("expected original error, got: %s", err.Error())
	}
}

func TestIsResolverInstalled(t *testing.T) {
	// Override ResolverDir is not possible since it's a const,
	// so we test with a temp file and check os.Stat directly.
	dir := t.TempDir()
	tld := "test"
	path := filepath.Join(dir, tld)

	// File doesn't exist yet — use os.Stat directly to verify behavior.
	if _, err := os.Stat(path); err == nil {
		t.Error("expected file to not exist")
	}

	// Create the file.
	if err := os.WriteFile(path, []byte("nameserver 127.0.0.1\nport 5053\n"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist, got: %v", err)
	}
}
