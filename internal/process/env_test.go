package process

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadEnvFile_Basic(t *testing.T) {
	path := writeEnvFile(t, "FOO=bar\nBAZ=qux\n")
	env, err := LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if env["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", env["FOO"], "bar")
	}
	if env["BAZ"] != "qux" {
		t.Errorf("BAZ = %q, want %q", env["BAZ"], "qux")
	}
}

func TestLoadEnvFile_CommentsAndBlanks(t *testing.T) {
	path := writeEnvFile(t, "# comment\n\nFOO=bar\n\n# another\nBAZ=qux\n")
	env, err := LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(env))
	}
}

func TestLoadEnvFile_QuotedValues(t *testing.T) {
	path := writeEnvFile(t, `FOO="hello world"
BAR='single quoted'
BAZ="has # hash"
`)
	env, err := LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if env["FOO"] != "hello world" {
		t.Errorf("FOO = %q, want %q", env["FOO"], "hello world")
	}
	if env["BAR"] != "single quoted" {
		t.Errorf("BAR = %q, want %q", env["BAR"], "single quoted")
	}
	if env["BAZ"] != "has # hash" {
		t.Errorf("BAZ = %q, want %q", env["BAZ"], "has # hash")
	}
}

func TestLoadEnvFile_InlineComment(t *testing.T) {
	path := writeEnvFile(t, "FOO=bar # this is a comment\n")
	env, err := LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if env["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", env["FOO"], "bar")
	}
}

func TestLoadEnvFile_Whitespace(t *testing.T) {
	path := writeEnvFile(t, "  FOO  =  bar  \n")
	env, err := LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if env["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", env["FOO"], "bar")
	}
}

func TestLoadEnvFile_EmptyValue(t *testing.T) {
	path := writeEnvFile(t, "FOO=\n")
	env, err := LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if env["FOO"] != "" {
		t.Errorf("FOO = %q, want empty", env["FOO"])
	}
}

func TestLoadEnvFile_NoEquals(t *testing.T) {
	path := writeEnvFile(t, "INVALID_LINE\n")
	_, err := LoadEnvFile(path)
	if err == nil {
		t.Fatal("expected error for line without '='")
	}
}

func TestLoadEnvFile_Missing(t *testing.T) {
	_, err := LoadEnvFile("/nonexistent/.env")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
