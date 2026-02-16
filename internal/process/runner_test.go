package process

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunner_BasicExecution(t *testing.T) {
	var mu sync.Mutex
	var lines []string

	r := &Runner{cfg: RunnerConfig{
		Name:    "test",
		Command: "echo hello",
		OnOutput: func(name, source, line string) {
			mu.Lock()
			lines = append(lines, line)
			mu.Unlock()
		},
	}}

	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case <-r.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("process did not exit in time")
	}

	if err := r.ExitErr(); err != nil {
		t.Fatalf("unexpected exit error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("expected [hello], got %v", lines)
	}
}

func TestRunner_StderrCapture(t *testing.T) {
	var mu sync.Mutex
	var sources []string

	r := &Runner{cfg: RunnerConfig{
		Name:    "test",
		Command: "echo out; echo err >&2",
		OnOutput: func(name, source, line string) {
			mu.Lock()
			sources = append(sources, source+":"+line)
			mu.Unlock()
		},
	}}

	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case <-r.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("process did not exit in time")
	}

	mu.Lock()
	defer mu.Unlock()

	var hasStdout, hasStderr bool
	for _, s := range sources {
		if strings.HasPrefix(s, "stdout:") {
			hasStdout = true
		}
		if strings.HasPrefix(s, "stderr:") {
			hasStderr = true
		}
	}
	if !hasStdout || !hasStderr {
		t.Errorf("expected both stdout and stderr, got %v", sources)
	}
}

func TestRunner_Stop(t *testing.T) {
	r := &Runner{cfg: RunnerConfig{
		Name:    "test",
		Command: "sleep 60",
	}}

	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := r.Stop(); err != nil {
		t.Fatalf("stop error: %v", err)
	}

	select {
	case <-r.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("process did not stop in time")
	}
}

func TestRunner_ExitError(t *testing.T) {
	r := &Runner{cfg: RunnerConfig{
		Name:    "test",
		Command: "exit 42",
	}}

	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case <-r.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("process did not exit in time")
	}

	if err := r.ExitErr(); err == nil {
		t.Fatal("expected exit error for non-zero exit code")
	}
}

func TestRunner_WorkingDirectory(t *testing.T) {
	dir := t.TempDir()

	var mu sync.Mutex
	var lines []string

	r := &Runner{cfg: RunnerConfig{
		Name:    "test",
		Command: "pwd",
		Dir:     dir,
		OnOutput: func(name, source, line string) {
			mu.Lock()
			lines = append(lines, line)
			mu.Unlock()
		},
	}}

	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case <-r.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(lines) == 0 || !strings.HasPrefix(lines[0], dir) {
		t.Errorf("expected working dir %s, got %v", dir, lines)
	}
}
