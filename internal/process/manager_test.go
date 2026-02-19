package process

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/httphatch/hatch/internal/config"
)

func testAppConfig(command string) config.Config {
	return config.Config{
		Projects: map[string]config.Project{
			"myapp": {
				Path:    "/tmp",
				Enabled: true,
				Services: map[string]config.Service{
					"worker": {Command: command},
				},
			},
		},
	}
}

// waitForRunning polls until the given service is running or deadline expires.
func waitForRunning(t *testing.T, mgr *Manager, id ServiceID, deadline time.Duration) {
	t.Helper()
	dl := time.After(deadline)
	for {
		st, ok := mgr.Statuses()[id]
		if ok && st.Running {
			return
		}
		select {
		case <-dl:
			t.Fatalf("timed out waiting for %v to be running", id)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestManager_StartStop(t *testing.T) {
	mgr := NewManager(ManagerConfig{})

	if err := mgr.ApplyConfig(testAppConfig("sleep 60")); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}
	waitForRunning(t, mgr, id, 5*time.Second)

	st := mgr.Statuses()[id]
	if st.Command != "sleep 60" {
		t.Errorf("command = %q, want %q", st.Command, "sleep 60")
	}

	if err := mgr.Stop(); err != nil {
		t.Fatal(err)
	}

	after := mgr.Statuses()
	if len(after) != 0 {
		t.Errorf("expected no processes after stop, got %d", len(after))
	}
}

func TestManager_SkipsDisabledProjects(t *testing.T) {
	cfg := config.Config{
		Projects: map[string]config.Project{
			"myapp": {
				Path:    "/tmp",
				Enabled: false,
				Services: map[string]config.Service{
					"worker": {Command: "sleep 60"},
				},
			},
		},
	}

	mgr := NewManager(ManagerConfig{})
	if err := mgr.ApplyConfig(cfg); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Stop() }()

	if len(mgr.Statuses()) != 0 {
		t.Error("expected no processes for disabled project")
	}
}

func TestManager_SkipsProxyOnlyServices(t *testing.T) {
	cfg := config.Config{
		Projects: map[string]config.Project{
			"myapp": {
				Path:    "/tmp",
				Enabled: true,
				Services: map[string]config.Service{
					"web": {Proxy: "http://localhost:3000"},
				},
			},
		},
	}

	mgr := NewManager(ManagerConfig{})
	if err := mgr.ApplyConfig(cfg); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Stop() }()

	if len(mgr.Statuses()) != 0 {
		t.Error("expected no managed processes for proxy-only service")
	}
}

func TestManager_ApplyConfigRemovesOld(t *testing.T) {
	mgr := NewManager(ManagerConfig{})

	if err := mgr.ApplyConfig(testAppConfig("sleep 60")); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}
	if _, ok := mgr.Statuses()[id]; !ok {
		t.Fatal("expected worker to be running")
	}

	// Apply empty config — should remove the process.
	if err := mgr.ApplyConfig(config.Config{Projects: map[string]config.Project{}}); err != nil {
		t.Fatal(err)
	}

	if len(mgr.Statuses()) != 0 {
		t.Error("expected no processes after removing from config")
	}
}

func TestManager_ApplyConfigRestartsChanged(t *testing.T) {
	mgr := NewManager(ManagerConfig{})
	defer func() { _ = mgr.Stop() }()

	if err := mgr.ApplyConfig(testAppConfig("sleep 60")); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}
	waitForRunning(t, mgr, id, 5*time.Second)
	st1 := mgr.Statuses()[id]

	// Change command — should restart.
	if err := mgr.ApplyConfig(testAppConfig("sleep 120")); err != nil {
		t.Fatal(err)
	}

	waitForRunning(t, mgr, id, 5*time.Second)
	st2 := mgr.Statuses()[id]
	if st2.Command != "sleep 120" {
		t.Errorf("command = %q, want %q", st2.Command, "sleep 120")
	}
	if !st2.StartedAt.After(st1.StartedAt) {
		t.Error("expected new start time after restart")
	}
}

func TestManager_OutputCallback(t *testing.T) {
	var mu sync.Mutex
	var output []string

	mgr := NewManager(ManagerConfig{
		OnOutput: func(project, service, source, line string) {
			mu.Lock()
			output = append(output, project+"/"+service+":"+source+":"+line)
			mu.Unlock()
		},
	})

	if err := mgr.ApplyConfig(testAppConfig("echo hello")); err != nil {
		t.Fatal(err)
	}

	// Wait for the output to be captured.
	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		n := len(output)
		mu.Unlock()
		if n > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for output")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	if err := mgr.Stop(); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(output) == 0 {
		t.Fatal("expected output callback to be called")
	}
}

func TestManager_ProcessRestartsOnCrash(t *testing.T) {
	mgr := NewManager(ManagerConfig{})
	defer func() { _ = mgr.Stop() }()

	// Use a command that exits immediately so it triggers a restart.
	if err := mgr.ApplyConfig(testAppConfig("exit 1")); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}

	// Wait for at least one restart.
	deadline := time.After(10 * time.Second)
	for {
		st, ok := mgr.Statuses()[id]
		if ok && st.Restarts >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for restart")
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestDeduplicateEnv(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "no duplicates",
			in:   []string{"A=1", "B=2"},
			want: []string{"A=1", "B=2"},
		},
		{
			name: "last value wins",
			in:   []string{"NODE_ENV=development", "FOO=bar", "NODE_ENV=production"},
			want: []string{"FOO=bar", "NODE_ENV=production"},
		},
		{
			name: "multiple duplicates",
			in:   []string{"A=1", "B=2", "A=3", "B=4"},
			want: []string{"A=3", "B=4"},
		},
		{
			name: "empty value preserved",
			in:   []string{"A=1", "A="},
			want: []string{"A="},
		},
		{
			name: "empty slice",
			in:   []string{},
			want: []string{},
		},
		{
			name: "entries without equals are dropped",
			in:   []string{"A=1", "MALFORMED", "B=2"},
			want: []string{"A=1", "B=2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicateEnv(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d; got %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestManager_StartedAtUpdatesOnRestart(t *testing.T) {
	mgr := NewManager(ManagerConfig{})
	defer func() { _ = mgr.Stop() }()

	// Use a command that exits immediately to trigger restarts.
	if err := mgr.ApplyConfig(testAppConfig("exit 0")); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}

	// Wait for at least one restart so we have a valid StartedAt.
	deadline := time.After(10 * time.Second)
	var firstStartedAt time.Time
	for {
		st, ok := mgr.Statuses()[id]
		if ok && st.Restarts >= 1 && !st.StartedAt.IsZero() {
			firstStartedAt = st.StartedAt
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for first restart")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Wait for another restart and verify StartedAt changed.
	deadline = time.After(10 * time.Second)
	for {
		st := mgr.Statuses()[id]
		if st.Restarts >= 2 && st.StartedAt.After(firstStartedAt) {
			return // success
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for StartedAt to update after restart")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestManager_StopProcess(t *testing.T) {
	mgr := NewManager(ManagerConfig{})
	defer func() { _ = mgr.Stop() }()

	if err := mgr.ApplyConfig(testAppConfig("sleep 60")); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}
	waitForRunning(t, mgr, id, 5*time.Second)

	if err := mgr.StopProcess(id); err != nil {
		t.Fatalf("StopProcess: %v", err)
	}

	st := mgr.Statuses()[id]
	if st.Running {
		t.Error("expected process to not be running after stop")
	}
	if !st.Stopped {
		t.Error("expected process to be marked as stopped")
	}

	// Verify it stays stopped (no auto-restart) by waiting a bit.
	time.Sleep(2 * time.Second)
	st = mgr.Statuses()[id]
	if st.Running {
		t.Error("expected process to remain stopped, but it restarted")
	}
}

func TestManager_StartProcess(t *testing.T) {
	mgr := NewManager(ManagerConfig{})
	defer func() { _ = mgr.Stop() }()

	if err := mgr.ApplyConfig(testAppConfig("sleep 60")); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}
	waitForRunning(t, mgr, id, 5*time.Second)

	if err := mgr.StopProcess(id); err != nil {
		t.Fatalf("StopProcess: %v", err)
	}

	if err := mgr.StartProcess(id); err != nil {
		t.Fatalf("StartProcess: %v", err)
	}

	waitForRunning(t, mgr, id, 5*time.Second)
	st := mgr.Statuses()[id]
	if st.Stopped {
		t.Error("expected process to not be marked as stopped after start")
	}
}

func TestManager_RestartProcess(t *testing.T) {
	mgr := NewManager(ManagerConfig{})
	defer func() { _ = mgr.Stop() }()

	if err := mgr.ApplyConfig(testAppConfig("sleep 60")); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}
	waitForRunning(t, mgr, id, 5*time.Second)
	st1 := mgr.Statuses()[id]

	if err := mgr.RestartProcess(id); err != nil {
		t.Fatalf("RestartProcess: %v", err)
	}

	waitForRunning(t, mgr, id, 5*time.Second)
	st2 := mgr.Statuses()[id]
	if !st2.StartedAt.After(st1.StartedAt) {
		t.Error("expected new start time after restart")
	}
}

func TestManager_RestartProcess_FromStopped(t *testing.T) {
	mgr := NewManager(ManagerConfig{})
	defer func() { _ = mgr.Stop() }()

	if err := mgr.ApplyConfig(testAppConfig("sleep 60")); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}
	waitForRunning(t, mgr, id, 5*time.Second)

	if err := mgr.StopProcess(id); err != nil {
		t.Fatalf("StopProcess: %v", err)
	}

	if err := mgr.RestartProcess(id); err != nil {
		t.Fatalf("RestartProcess from stopped: %v", err)
	}

	waitForRunning(t, mgr, id, 5*time.Second)
	st := mgr.Statuses()[id]
	if !st.Running {
		t.Error("expected process to be running after restart from stopped")
	}
}

func TestManager_StopProcess_NotFound(t *testing.T) {
	mgr := NewManager(ManagerConfig{})

	id := ServiceID{Project: "unknown", Service: "svc"}
	err := mgr.StopProcess(id)
	if err == nil {
		t.Fatal("expected error for unknown ServiceID")
	}
	if !errors.Is(err, ErrProcessNotFound) {
		t.Errorf("expected ErrProcessNotFound, got: %v", err)
	}
}

func TestManager_StartProcess_AlreadyRunning(t *testing.T) {
	mgr := NewManager(ManagerConfig{})
	defer func() { _ = mgr.Stop() }()

	if err := mgr.ApplyConfig(testAppConfig("sleep 60")); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}
	waitForRunning(t, mgr, id, 5*time.Second)

	err := mgr.StartProcess(id)
	if err == nil {
		t.Fatal("expected error when starting an already running process")
	}
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("expected ErrAlreadyRunning, got: %v", err)
	}
}

func TestManager_SkipsMissingPath(t *testing.T) {
	cfg := config.Config{
		Projects: map[string]config.Project{
			"myapp": {
				Path:    "/nonexistent/path/that/does/not/exist",
				Enabled: true,
				Services: map[string]config.Service{
					"worker": {Command: "sleep 60"},
				},
			},
		},
	}

	mgr := NewManager(ManagerConfig{})
	if err := mgr.ApplyConfig(cfg); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Stop() }()

	if len(mgr.Statuses()) != 0 {
		t.Error("expected no managed processes for project with missing path")
	}
}

func TestManager_NoopForUnchanged(t *testing.T) {
	mgr := NewManager(ManagerConfig{})
	defer func() { _ = mgr.Stop() }()

	cfg := testAppConfig("sleep 60")
	if err := mgr.ApplyConfig(cfg); err != nil {
		t.Fatal(err)
	}

	id := ServiceID{Project: "myapp", Service: "worker"}
	waitForRunning(t, mgr, id, 5*time.Second)
	st1 := mgr.Statuses()[id]

	// Apply same config again — should be a no-op.
	if err := mgr.ApplyConfig(cfg); err != nil {
		t.Fatal(err)
	}

	st2 := mgr.Statuses()[id]
	if st2.StartedAt != st1.StartedAt {
		t.Error("expected start time to remain the same for unchanged config")
	}
}
