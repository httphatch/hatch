package health

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/httphatch/hatch/internal/config"
)

// --- helpers ---

// startTestListener opens a TCP listener on an ephemeral port, accepts
// connections in the background, and closes everything on test cleanup.
func startTestListener(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	return ln
}

// testConfig builds a minimal config.Config with one enabled project/service
// pointing at the given address.
func testConfig(addr string) config.Config {
	return config.Config{
		Projects: map[string]config.Project{
			"myproject": {
				Enabled: true,
				Services: map[string]config.Service{
					"web": {Proxy: "http://" + addr},
				},
			},
		},
	}
}

// waitForStatus polls the checker until the given key reaches the expected
// status or the deadline expires.
func waitForStatus(t *testing.T, c *Checker, key ServiceKey, expected Status, deadline time.Duration) {
	t.Helper()
	dl := time.After(deadline)
	for {
		select {
		case <-dl:
			ss, ok := c.ServiceStatus(key)
			if !ok {
				t.Fatalf("timed out: key %v not found", key)
			}
			t.Fatalf("timed out waiting for %v: got %v", expected, ss.Status)
		default:
			ss, ok := c.ServiceStatus(key)
			if ok && ss.Status == expected {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func newTestChecker(opts ...func(*CheckerConfig)) *Checker {
	cfg := CheckerConfig{
		Interval: 50 * time.Millisecond,
		Timeout:  100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return NewChecker(cfg)
}

var testKey = ServiceKey{Project: "myproject", Service: "web"}

// --- tests ---

func TestChecker_StartStop(t *testing.T) {
	ln := startTestListener(t)
	c := newTestChecker()

	if err := c.Start(testConfig(ln.Addr().String())); err != nil {
		t.Fatal(err)
	}
	if err := c.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestChecker_DoubleStartFails(t *testing.T) {
	ln := startTestListener(t)
	c := newTestChecker()

	if err := c.Start(testConfig(ln.Addr().String())); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop() }()

	if err := c.Start(testConfig(ln.Addr().String())); err == nil {
		t.Fatal("expected error on double start")
	}
}

func TestChecker_StopWithoutStart(t *testing.T) {
	c := newTestChecker()
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop on idle checker should not error: %v", err)
	}
}

func TestChecker_HealthyService(t *testing.T) {
	ln := startTestListener(t)
	c := newTestChecker()

	if err := c.Start(testConfig(ln.Addr().String())); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop() }()

	waitForStatus(t, c, testKey, StatusHealthy, 2*time.Second)

	ss, _ := c.ServiceStatus(testKey)
	if ss.Addr != ln.Addr().String() {
		t.Fatalf("addr = %q, want %q", ss.Addr, ln.Addr().String())
	}
	if ss.Since.IsZero() {
		t.Fatal("Since should not be zero")
	}
	if ss.LastCheck.IsZero() {
		t.Fatal("LastCheck should not be zero")
	}
}

func TestChecker_UnhealthyService(t *testing.T) {
	// Pick a port that nothing is listening on.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close() // close immediately so nothing is listening

	c := newTestChecker()
	if err := c.Start(testConfig(addr)); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop() }()

	waitForStatus(t, c, testKey, StatusUnhealthy, 2*time.Second)
}

func TestChecker_ServiceComesOnline(t *testing.T) {
	// Grab an ephemeral port, close the listener so the service looks down.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	c := newTestChecker()
	if err := c.Start(testConfig(addr)); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop() }()

	waitForStatus(t, c, testKey, StatusUnhealthy, 2*time.Second)

	// Start a listener on the same port.
	ln2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Skipf("could not re-listen on %s: %v", addr, err)
	}
	defer func() { _ = ln2.Close() }()
	go func() {
		for {
			conn, err := ln2.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	waitForStatus(t, c, testKey, StatusHealthy, 2*time.Second)
}

func TestChecker_ServiceGoesOffline(t *testing.T) {
	ln := startTestListener(t)
	addr := ln.Addr().String()

	c := newTestChecker()
	if err := c.Start(testConfig(addr)); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop() }()

	waitForStatus(t, c, testKey, StatusHealthy, 2*time.Second)

	_ = ln.Close()

	waitForStatus(t, c, testKey, StatusUnhealthy, 2*time.Second)
}

func TestChecker_OnChangeCallback(t *testing.T) {
	ln := startTestListener(t)

	var mu sync.Mutex
	var transitions []struct{ from, to Status }

	c := newTestChecker(func(cfg *CheckerConfig) {
		cfg.OnChange = func(key ServiceKey, from, to Status) {
			mu.Lock()
			transitions = append(transitions, struct{ from, to Status }{from, to})
			mu.Unlock()
		}
	})

	if err := c.Start(testConfig(ln.Addr().String())); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop() }()

	waitForStatus(t, c, testKey, StatusHealthy, 2*time.Second)

	mu.Lock()
	count := len(transitions)
	mu.Unlock()

	if count == 0 {
		t.Fatal("expected at least one transition callback")
	}

	mu.Lock()
	first := transitions[0]
	mu.Unlock()

	if first.from != StatusUnknown || first.to != StatusHealthy {
		t.Fatalf("first transition = %v→%v, want unknown→healthy", first.from, first.to)
	}
}

func TestChecker_UpdateConfig(t *testing.T) {
	ln1 := startTestListener(t)
	ln2 := startTestListener(t)

	c := newTestChecker()
	if err := c.Start(testConfig(ln1.Addr().String())); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop() }()

	waitForStatus(t, c, testKey, StatusHealthy, 2*time.Second)

	// Update config: replace "web" service and add a second project.
	newCfg := config.Config{
		Projects: map[string]config.Project{
			"myproject": {
				Enabled: true,
				Services: map[string]config.Service{
					"api": {Proxy: "http://" + ln2.Addr().String()},
				},
			},
		},
	}
	c.UpdateConfig(newCfg)

	apiKey := ServiceKey{Project: "myproject", Service: "api"}
	waitForStatus(t, c, apiKey, StatusHealthy, 2*time.Second)

	// Old "web" service should be gone.
	if _, ok := c.ServiceStatus(testKey); ok {
		t.Fatal("old service 'web' should have been removed")
	}
}

func TestChecker_DisabledProjectSkipped(t *testing.T) {
	ln := startTestListener(t)

	cfg := config.Config{
		Projects: map[string]config.Project{
			"disabled": {
				Enabled: false,
				Services: map[string]config.Service{
					"web": {Proxy: "http://" + ln.Addr().String()},
				},
			},
		},
	}

	c := newTestChecker()
	if err := c.Start(cfg); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop() }()

	// Give a tick for the checker to run.
	time.Sleep(150 * time.Millisecond)

	all := c.ServiceStatuses()
	if len(all) != 0 {
		t.Fatalf("expected no targets for disabled project, got %d", len(all))
	}
}

func TestChecker_CommandOnlyServiceSkipped(t *testing.T) {
	ln := startTestListener(t)

	cfg := config.Config{
		Projects: map[string]config.Project{
			"myproject": {
				Enabled: true,
				Services: map[string]config.Service{
					"web":    {Proxy: "http://" + ln.Addr().String()},
					"worker": {Command: "php artisan queue:work"},
				},
			},
		},
	}

	c := newTestChecker()
	if err := c.Start(cfg); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop() }()

	waitForStatus(t, c, testKey, StatusHealthy, 2*time.Second)

	all := c.ServiceStatuses()
	if len(all) != 1 {
		t.Fatalf("expected 1 target (command-only skipped), got %d", len(all))
	}
	workerKey := ServiceKey{Project: "myproject", Service: "worker"}
	if _, ok := all[workerKey]; ok {
		t.Fatal("command-only service should not be tracked by health checker")
	}
}

func TestExtractDialAddress(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"http://localhost:3000", "localhost:3000"},
		{"http://127.0.0.1:8080", "127.0.0.1:8080"},
		{"https://localhost:443", "localhost:443"},
		{"http://localhost", "localhost:80"},
		{"https://localhost", "localhost:443"},
		{"http://[::1]:9000", "[::1]:9000"},
		{"://bad", "://bad"}, // unparseable → returned as-is
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractDialAddress(tt.input)
			if got != tt.want {
				t.Fatalf("extractDialAddress(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
