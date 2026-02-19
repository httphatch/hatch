package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/health"
	"github.com/httphatch/hatch/internal/process"
)

// DaemonControl allows the API to trigger daemon operations.
type DaemonControl interface {
	ReloadConfig() error
}

// DashboardShower can show the GUI dashboard window.
type DashboardShower interface {
	ShowDashboard()
}

// ProcessController provides process status and control operations.
type ProcessController interface {
	Statuses() map[process.ServiceID]process.ProcessStatus
	StopProcess(id process.ServiceID) error
	StartProcess(id process.ServiceID) error
	RestartProcess(id process.ServiceID) error
}

// Server is the HTTP API server for the Hatch dashboard.
type Server struct {
	httpSrv   *http.Server
	daemon    DaemonControl
	dashboard DashboardShower
	health    *health.Checker
	processes ProcessController
	caPaths   certs.CAPaths
	version   string
	startTime time.Time
	logHub    *LogHub
	outputHub *ProcessOutputHub
	cfgMu     sync.Mutex // serializes config read-modify-write operations
}

// ServerConfig holds the configuration for creating a new API server.
type ServerConfig struct {
	Addr      string
	Health    *health.Checker
	Processes ProcessController
	Daemon    DaemonControl
	Dashboard DashboardShower
	CAPaths   certs.CAPaths
	Version   string
	StartTime time.Time
	LogHub    *LogHub
	OutputHub *ProcessOutputHub
}

// NewServer creates a new API server with the given configuration.
func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		daemon:    cfg.Daemon,
		dashboard: cfg.Dashboard,
		health:    cfg.Health,
		processes: cfg.Processes,
		caPaths:   cfg.CAPaths,
		version:   cfg.Version,
		startTime: cfg.StartTime,
		logHub:    cfg.LogHub,
		outputHub: cfg.OutputHub,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpSrv = &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return s
}

// Start begins serving HTTP requests in a background goroutine.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.httpSrv.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.httpSrv.Addr, err)
	}

	go func() {
		if err := s.httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("api server error")
		}
	}()

	return nil
}

// Handler returns the server's HTTP handler for in-process use (e.g. the
// Wails asset handler can call it directly without a network round-trip).
func (s *Server) Handler() http.Handler {
	return s.httpSrv.Handler
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/projects", s.handleListProjects)
	mux.HandleFunc("POST /api/projects", requireJSON(s.handleAddProject))
	mux.HandleFunc("PUT /api/projects/{name}", requireJSON(s.handleUpdateProject))
	mux.HandleFunc("DELETE /api/projects/{name}", s.handleDeleteProject)
	mux.HandleFunc("PATCH /api/projects/{name}/toggle", s.handleToggleProject)
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/processes", s.handleProcesses)
	mux.HandleFunc("POST /api/processes/{project}/{service}/stop", requireJSON(s.handleProcessStop))
	mux.HandleFunc("POST /api/processes/{project}/{service}/start", requireJSON(s.handleProcessStart))
	mux.HandleFunc("POST /api/processes/{project}/{service}/restart", requireJSON(s.handleProcessRestart))
	mux.HandleFunc("GET /api/processes/output", s.handleProcessOutput)
	mux.HandleFunc("GET /api/logs", s.handleLogs)
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("PUT /api/config", s.handlePutConfig)
	mux.HandleFunc("GET /api/certs", s.handleCerts)
	mux.HandleFunc("POST /api/restart", requireJSON(s.handleRestart))
	mux.HandleFunc("POST /api/dashboard/show", requireJSON(s.handleShowDashboard))
}
