package process

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestCleanOrphans_NoFile(t *testing.T) {
	if err := CleanOrphans("/nonexistent/processes.json"); err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
}

func TestCleanOrphans_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "processes.json")

	if err := os.WriteFile(pidFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CleanOrphans(pidFile); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("expected PID file to be removed after cleanup")
	}
}

func TestCleanOrphans_CorruptedFile(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "processes.json")

	if err := os.WriteFile(pidFile, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CleanOrphans(pidFile); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("expected PID file to be removed after cleanup")
	}
}

func TestCleanOrphans_StaleProcess(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "processes.json")

	// Use a PID that almost certainly doesn't exist.
	pids := map[string]int{"myapp/web": 99999999}
	data, _ := json.Marshal(pids)
	if err := os.WriteFile(pidFile, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CleanOrphans(pidFile); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("expected PID file to be removed after cleanup")
	}
}

func TestCleanOrphans_KillsRunningProcess(t *testing.T) {
	// Start a process that we can kill.
	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Reap the zombie in a background goroutine so Signal(0) checks work.
	waitDone := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(waitDone)
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-waitDone
	})

	pid := cmd.Process.Pid

	dir := t.TempDir()
	pidFile := filepath.Join(dir, "processes.json")

	pids := map[string]int{"myapp/web": pid}
	data, _ := json.Marshal(pids)
	if err := os.WriteFile(pidFile, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CleanOrphans(pidFile); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the process was killed by waiting for cmd.Wait() to return.
	select {
	case <-waitDone:
		// Process was killed successfully.
	case <-time.After(3 * time.Second):
		t.Fatal("orphaned process was not killed in time")
	}
}

func TestSavePIDs_WritesAndCleans(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "processes.json")

	mgr := NewManager(ManagerConfig{PIDFile: pidFile})
	if err := mgr.ApplyConfig(testAppConfig("sleep 60")); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}
	waitForRunning(t, mgr, id, 5*time.Second)

	// Verify PID file was written.
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("expected PID file to exist: %v", err)
	}

	var pids map[string]int
	if err := json.Unmarshal(data, &pids); err != nil {
		t.Fatalf("failed to parse PID file: %v", err)
	}

	if pid, ok := pids["myapp/worker"]; !ok || pid == 0 {
		t.Errorf("expected non-zero PID for myapp/worker, got %v", pids)
	}

	// Stop should remove the PID file.
	if err := mgr.Stop(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("expected PID file to be removed after Stop()")
	}
}
