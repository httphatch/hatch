package daemon

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePlist_KeepAliveTrue(t *testing.T) {
	data, err := GeneratePlist(LaunchdConfig{
		Label:            PlistLabel,
		BinaryPath:       "/usr/local/bin/hatch",
		WorkingDirectory: "/tmp",
		LogDir:           "/tmp",
		KeepAlive:        true,
	})
	if err != nil {
		t.Fatalf("GeneratePlist() error: %v", err)
	}

	s := string(data)

	// KeepAlive and RunAtLoad should both be true.
	if !containsKeyValue(s, "KeepAlive", "<true/>") {
		t.Error("expected KeepAlive <true/>")
	}
	if !containsKeyValue(s, "RunAtLoad", "<true/>") {
		t.Error("expected RunAtLoad <true/>")
	}
}

func TestGeneratePlist_KeepAliveFalse(t *testing.T) {
	data, err := GeneratePlist(LaunchdConfig{
		Label:            PlistLabel,
		BinaryPath:       "/usr/local/bin/hatch",
		WorkingDirectory: "/tmp",
		LogDir:           "/tmp",
		KeepAlive:        false,
	})
	if err != nil {
		t.Fatalf("GeneratePlist() error: %v", err)
	}

	s := string(data)

	if !containsKeyValue(s, "KeepAlive", "<false/>") {
		t.Error("expected KeepAlive <false/>")
	}
	if !containsKeyValue(s, "RunAtLoad", "<false/>") {
		t.Error("expected RunAtLoad <false/>")
	}
}

func TestGeneratePlist_ContainsRunArg(t *testing.T) {
	data, err := GeneratePlist(LaunchdConfig{
		Label:            PlistLabel,
		BinaryPath:       "/usr/local/bin/hatch",
		WorkingDirectory: "/tmp",
		LogDir:           "/tmp",
	})
	if err != nil {
		t.Fatalf("GeneratePlist() error: %v", err)
	}

	if !strings.Contains(string(data), "<string>_run</string>") {
		t.Error("plist should contain _run in ProgramArguments")
	}
}

func TestGeneratePlist_BinaryPath(t *testing.T) {
	binPath := "/opt/hatch/bin/hatch"
	data, err := GeneratePlist(LaunchdConfig{
		Label:            PlistLabel,
		BinaryPath:       binPath,
		WorkingDirectory: "/tmp",
		LogDir:           "/tmp",
	})
	if err != nil {
		t.Fatalf("GeneratePlist() error: %v", err)
	}

	if !strings.Contains(string(data), "<string>"+binPath+"</string>") {
		t.Errorf("plist should contain binary path %q", binPath)
	}
}

func TestGeneratePlist_ValidXML(t *testing.T) {
	data, err := GeneratePlist(LaunchdConfig{
		Label:            PlistLabel,
		BinaryPath:       "/usr/local/bin/hatch",
		WorkingDirectory: "/tmp",
		LogDir:           "/tmp",
		KeepAlive:        true,
	})
	if err != nil {
		t.Fatalf("GeneratePlist() error: %v", err)
	}

	// encoding/xml should be able to parse it as valid XML.
	var v any
	if err := xml.Unmarshal(data, &v); err != nil {
		t.Errorf("generated plist is not valid XML: %v\n%s", err, data)
	}
}

func TestPlistPath(t *testing.T) {
	path, err := PlistPath()
	if err != nil {
		t.Fatalf("PlistPath() error: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "Library", "LaunchAgents", PlistLabel+".plist")
	if path != expected {
		t.Errorf("PlistPath() = %q, want %q", path, expected)
	}
}

// containsKeyValue checks if the plist XML contains a <key>key</key> followed
// (within a few lines) by the given value string.
func containsKeyValue(plist, key, value string) bool {
	lines := strings.Split(plist, "\n")
	for i, line := range lines {
		if strings.Contains(line, "<key>"+key+"</key>") {
			// Check the next line for the value.
			if i+1 < len(lines) && strings.Contains(strings.TrimSpace(lines[i+1]), value) {
				return true
			}
		}
	}
	return false
}
