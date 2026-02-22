package api

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/process"
	"github.com/httphatch/hatch/internal/tunnel"
)

// maxBodySize is the maximum allowed request body (1 MB).
const maxBodySize = 1 << 20

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// requireJSON rejects requests that don't have Content-Type: application/json.
// This also serves as CSRF protection since non-simple content types trigger
// a CORS preflight that this server does not respond to (CORS is only handled
// by the Caddy proxy layer for .test domains, not by the API server).
func requireJSON(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
			return
		}
		next(w, r)
	}
}

func limitBody(r *http.Request, w http.ResponseWriter) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"pid":     os.Getpid(),
		"uptime":  time.Since(s.startTime).Truncate(time.Second).String(),
		"version": s.version,
	})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadWithProjectConfigs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}
	projects := cfg.Projects
	if projects == nil {
		projects = make(map[string]config.Project)
	}
	// Strip tokens from API response to avoid leaking secrets.
	for name, proj := range projects {
		proj.CloudflareToken = ""
		projects[name] = proj
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleAddProject(w http.ResponseWriter, r *http.Request) {
	limitBody(r, w)

	var req struct {
		Name    string         `json:"name"`
		Project config.Project `json:"project"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	cfg, err := config.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}
	if _, exists := cfg.Projects[req.Name]; exists {
		writeError(w, http.StatusConflict, fmt.Sprintf("project %q already exists", req.Name))
		return
	}

	if cfg.Projects == nil {
		cfg.Projects = make(map[string]config.Project)
	}
	cfg.Projects[req.Name] = req.Project

	if errs := config.Validate(cfg); len(errs) > 0 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid config: %v", errs))
		return
	}
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config")
		return
	}
	req.Project.CloudflareToken = ""
	writeJSON(w, http.StatusCreated, req.Project)
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	limitBody(r, w)
	name := r.PathValue("name")

	var proj config.Project
	if err := json.NewDecoder(r.Body).Decode(&proj); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	cfg, err := config.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}
	if _, exists := cfg.Projects[name]; !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %q not found", name))
		return
	}

	cfg.Projects[name] = proj
	if errs := config.Validate(cfg); len(errs) > 0 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid config: %v", errs))
		return
	}
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config")
		return
	}
	proj.CloudflareToken = ""
	writeJSON(w, http.StatusOK, proj)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	cfg, err := config.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}
	if _, exists := cfg.Projects[name]; !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %q not found", name))
		return
	}

	delete(cfg.Projects, name)
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleToggleProject(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	cfg, err := config.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}

	proj, exists := cfg.Projects[name]
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %q not found", name))
		return
	}

	proj.Enabled = !proj.Enabled
	cfg.Projects[name] = proj

	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": proj.Enabled})
}

func (s *Server) handleProcesses(w http.ResponseWriter, r *http.Request) {
	if s.processes == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	statuses := s.processes.Statuses()

	type processInfo struct {
		Project   string `json:"project"`
		Service   string `json:"service"`
		Command   string `json:"command"`
		Running   bool   `json:"running"`
		Stopped   bool   `json:"stopped"`
		Restarts  int    `json:"restarts"`
		StartedAt string `json:"started_at"`
	}

	result := make([]processInfo, 0, len(statuses))
	for id, st := range statuses {
		result = append(result, processInfo{
			Project:   id.Project,
			Service:   id.Service,
			Command:   st.Command,
			Running:   st.Running,
			Stopped:   st.Stopped,
			Restarts:  st.Restarts,
			StartedAt: st.StartedAt.Format(time.RFC3339),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Project != result[j].Project {
			return result[i].Project < result[j].Project
		}
		return result[i].Service < result[j].Service
	})
	writeJSON(w, http.StatusOK, result)
}

func processErrorStatus(err error) int {
	switch {
	case errors.Is(err, process.ErrProcessNotFound):
		return http.StatusNotFound
	case errors.Is(err, process.ErrAlreadyStopped), errors.Is(err, process.ErrAlreadyRunning):
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

func tunnelErrorStatus(err error) int {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"):
		return http.StatusNotFound
	case strings.Contains(msg, "already running"), strings.Contains(msg, "already stopped"):
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

func (s *Server) handleProcessStop(w http.ResponseWriter, r *http.Request) {
	if s.processes == nil {
		writeError(w, http.StatusNotImplemented, "process control not available")
		return
	}
	id := process.ServiceID{
		Project: r.PathValue("project"),
		Service: r.PathValue("service"),
	}
	if err := s.processes.StopProcess(id); err != nil {
		writeError(w, processErrorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) handleProcessStart(w http.ResponseWriter, r *http.Request) {
	if s.processes == nil {
		writeError(w, http.StatusNotImplemented, "process control not available")
		return
	}
	id := process.ServiceID{
		Project: r.PathValue("project"),
		Service: r.PathValue("service"),
	}
	if err := s.processes.StartProcess(id); err != nil {
		writeError(w, processErrorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Server) handleProcessRestart(w http.ResponseWriter, r *http.Request) {
	if s.processes == nil {
		writeError(w, http.StatusNotImplemented, "process control not available")
		return
	}
	id := process.ServiceID{
		Project: r.PathValue("project"),
		Service: r.PathValue("service"),
	}
	if err := s.processes.RestartProcess(id); err != nil {
		writeError(w, processErrorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}

func (s *Server) handleProcessOutput(w http.ResponseWriter, r *http.Request) {
	if s.outputHub == nil {
		writeError(w, http.StatusServiceUnavailable, "process output not available")
		return
	}

	project := r.URL.Query().Get("project")
	service := r.URL.Query().Get("service")
	if project == "" || service == "" {
		writeError(w, http.StatusBadRequest, "project and service query parameters are required")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	ch, cleanup := s.outputHub.Subscribe(project, service)
	defer cleanup()

	for {
		select {
		case <-r.Context().Done():
			return
		case line := <-ch:
			data, err := json.Marshal(line)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	statuses := s.health.ServiceStatuses()

	type serviceHealth struct {
		Project   string `json:"project"`
		Service   string `json:"service"`
		Status    string `json:"status"`
		Addr      string `json:"addr"`
		Since     string `json:"since"`
		LastCheck string `json:"last_check"`
	}

	result := make([]serviceHealth, 0, len(statuses))
	for key, st := range statuses {
		result = append(result, serviceHealth{
			Project:   key.Project,
			Service:   key.Service,
			Status:    st.Status.String(),
			Addr:      st.Addr,
			Since:     st.Since.Format(time.RFC3339),
			LastCheck: st.LastCheck.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	ch, cleanup := s.logHub.Subscribe()
	defer cleanup()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			_, _ = fmt.Fprintf(w, "data: %s\n\n", bytes.TrimRight(msg, "\n"))
			flusher.Flush()
		}
	}
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(config.ConfigFile())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read config")
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(data)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	// The non-simple Content-Type also serves as CSRF protection: it triggers
	// a CORS preflight that this server does not respond to.
	if ct := r.Header.Get("Content-Type"); ct != "application/yaml" {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/yaml")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	cfg := config.DefaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid YAML: %s", err))
		return
	}
	if errs := config.Validate(cfg); len(errs) > 0 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid config: %v", errs))
		return
	}

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	if err := s.daemon.ReloadConfig(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload config")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}

func (s *Server) handleShowDashboard(w http.ResponseWriter, r *http.Request) {
	if s.dashboard == nil {
		writeError(w, http.StatusNotImplemented, "dashboard not available on this platform")
		return
	}
	s.dashboard.ShowDashboard()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleTunnels(w http.ResponseWriter, r *http.Request) {
	if s.tunnels == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	statuses := s.tunnels.Statuses()

	type tunnelInfo struct {
		Project   string `json:"project"`
		Service   string `json:"service"`
		Running   bool   `json:"running"`
		Starting  bool   `json:"starting"`
		URL       string `json:"url"`
		Type      string `json:"type"`
		StartedAt string `json:"started_at"`
		Error     string `json:"error,omitempty"`
	}

	result := make([]tunnelInfo, 0, len(statuses))
	for id, st := range statuses {
		startedAt := ""
		if !st.StartedAt.IsZero() {
			startedAt = st.StartedAt.Format(time.RFC3339)
		}
		result = append(result, tunnelInfo{
			Project:   id.Project,
			Service:   id.Service,
			Running:   st.Running,
			Starting:  st.Starting,
			URL:       st.URL,
			Type:      st.Type,
			StartedAt: startedAt,
			Error:     st.Error,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Project != result[j].Project {
			return result[i].Project < result[j].Project
		}
		return result[i].Service < result[j].Service
	})
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleTunnelStart(w http.ResponseWriter, r *http.Request) {
	limitBody(r, w)
	if s.tunnels == nil {
		writeError(w, http.StatusNotImplemented, "tunnel control not available")
		return
	}

	project := r.PathValue("project")
	service := r.PathValue("service")

	cfg, err := config.LoadWithProjectConfigs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}

	proj, ok := cfg.Projects[project]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %q not found", project))
		return
	}

	svc, ok := proj.Services[service]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("service %q not found in project %q", service, project))
		return
	}

	if svc.Proxy == "" {
		writeError(w, http.StatusBadRequest, "service has no proxy (nothing to tunnel)")
		return
	}

	tunnelVal := string(svc.Tunnel)
	if tunnelVal == "" {
		tunnelVal = "true"
	}

	token := proj.CloudflareToken
	if token == "" {
		token = cfg.Settings.CloudflareToken
	}

	upstream := svc.Proxy
	id := tunnel.TunnelID{Project: project, Service: service}
	if err := s.tunnels.StartTunnel(id, upstream, tunnelVal, token, cfg.Settings.CloudflareAccountID); err != nil {
		writeError(w, tunnelErrorStatus(err), err.Error())
		return
	}

	statuses := s.tunnels.Statuses()
	if st, ok := statuses[id]; ok {
		writeJSON(w, http.StatusOK, map[string]string{"url": st.URL, "status": "started"})
	} else {
		writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
	}
}

func (s *Server) handleTunnelStop(w http.ResponseWriter, r *http.Request) {
	limitBody(r, w)
	if s.tunnels == nil {
		writeError(w, http.StatusNotImplemented, "tunnel control not available")
		return
	}

	id := tunnel.TunnelID{
		Project: r.PathValue("project"),
		Service: r.PathValue("service"),
	}
	if err := s.tunnels.StopTunnel(id); err != nil {
		writeError(w, tunnelErrorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}
	cfg.Settings.CloudflareToken = ""
	writeJSON(w, http.StatusOK, cfg.Settings)
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	limitBody(r, w)

	var settings config.Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	cfg, err := config.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}

	// Preserve existing token when the submitted value is empty (the GET
	// endpoint redacts it, so the dashboard sends back an empty string).
	if settings.CloudflareToken == "" {
		settings.CloudflareToken = cfg.Settings.CloudflareToken
	}

	cfg.Settings = settings

	if errs := config.Validate(cfg); len(errs) > 0 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid config: %v", errs))
		return
	}
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCerts(w http.ResponseWriter, r *http.Request) {
	type certInfo struct {
		Exists  bool   `json:"exists"`
		Subject string `json:"subject,omitempty"`
		NotAfter string `json:"not_after,omitempty"`
		Trusted *bool  `json:"trusted,omitempty"`
	}

	parseCert := func(path string) *x509.Certificate {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		block, _ := pem.Decode(data)
		if block == nil {
			return nil
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil
		}
		return cert
	}

	var rootCA, intermediateCA certInfo

	if certs.CAExists(s.caPaths) {
		rootCA.Exists = true
		if cert := parseCert(s.caPaths.Cert); cert != nil {
			rootCA.Subject = cert.Subject.CommonName
			rootCA.NotAfter = cert.NotAfter.Format(time.RFC3339)
		}
		trusted := certs.IsCATrusted(s.caPaths.Cert)
		rootCA.Trusted = &trusted
	}

	if certs.IntermediateCAExists(s.caPaths) {
		intermediateCA.Exists = true
		if cert := parseCert(s.caPaths.IntermediateCert); cert != nil {
			intermediateCA.Subject = cert.Subject.CommonName
			intermediateCA.NotAfter = cert.NotAfter.Format(time.RFC3339)
		}
	}

	writeJSON(w, http.StatusOK, map[string]certInfo{
		"root_ca":         rootCA,
		"intermediate_ca": intermediateCA,
	})
}
