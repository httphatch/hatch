package daemon

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestWritePID_CreatesFile(t *testing.T) {
	t.Setenv("HATCH_HOME", t.TempDir())

	f, err := WritePID()
	if err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}
	defer func() { _ = RemovePID(f) }()

	data, err := os.ReadFile(PIDFile())
	if err != nil {
		t.Fatalf("read pid file: %v", err)
	}

	got, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parse pid: %v", err)
	}
	if got != os.Getpid() {
		t.Errorf("PID = %d, want %d", got, os.Getpid())
	}
}

func TestWritePID_PreventsDoubleRun(t *testing.T) {
	t.Setenv("HATCH_HOME", t.TempDir())

	f, err := WritePID()
	if err != nil {
		t.Fatalf("first WritePID() error: %v", err)
	}
	defer func() { _ = RemovePID(f) }()

	_, err = WritePID()
	if err == nil {
		t.Fatal("second WritePID() should have returned an error")
	}
}

func TestRemovePID_CleansUp(t *testing.T) {
	t.Setenv("HATCH_HOME", t.TempDir())

	f, err := WritePID()
	if err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}

	if err := RemovePID(f); err != nil {
		t.Fatalf("RemovePID() error: %v", err)
	}

	if _, err := os.Stat(PIDFile()); !os.IsNotExist(err) {
		t.Error("PID file still exists after RemovePID")
	}

	// Lock should be released — new WritePID should succeed.
	f2, err := WritePID()
	if err != nil {
		t.Fatalf("WritePID() after RemovePID error: %v", err)
	}
	defer func() { _ = RemovePID(f2) }()
}

func TestIsRunning_NoFile(t *testing.T) {
	t.Setenv("HATCH_HOME", t.TempDir())

	running, pid, err := IsRunning()
	if err != nil {
		t.Fatalf("IsRunning() error: %v", err)
	}
	if running {
		t.Error("IsRunning() = true, want false (no file)")
	}
	if pid != 0 {
		t.Errorf("pid = %d, want 0", pid)
	}
}

func TestIsRunning_LockedFile(t *testing.T) {
	t.Setenv("HATCH_HOME", t.TempDir())

	f, err := WritePID()
	if err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}
	defer func() { _ = RemovePID(f) }()

	running, pid, err := IsRunning()
	if err != nil {
		t.Fatalf("IsRunning() error: %v", err)
	}
	if !running {
		t.Error("IsRunning() = false, want true")
	}
	if pid != os.Getpid() {
		t.Errorf("pid = %d, want %d", pid, os.Getpid())
	}
}

func TestIsRunning_StaleFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HATCH_HOME", dir)

	// Write a PID file without holding the lock.
	if err := os.WriteFile(PIDFile(), []byte("99999\n"), 0o644); err != nil {
		t.Fatalf("write stale pid: %v", err)
	}

	running, _, err := IsRunning()
	if err != nil {
		t.Fatalf("IsRunning() error: %v", err)
	}
	if running {
		t.Error("IsRunning() = true, want false (stale file)")
	}
}

func TestReadPID(t *testing.T) {
	t.Setenv("HATCH_HOME", t.TempDir())

	f, err := WritePID()
	if err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}
	defer func() { _ = RemovePID(f) }()

	pid, err := ReadPID()
	if err != nil {
		t.Fatalf("ReadPID() error: %v", err)
	}
	if pid != os.Getpid() {
		t.Errorf("ReadPID() = %d, want %d", pid, os.Getpid())
	}
}
