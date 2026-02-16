package api

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/config"
)

// maxBodySize is the maximum allowed request body (1 MB).
const maxBodySize = 1 << 20

// corsLocal wraps a handler to allow cross-origin requests from the Wails
// webview (which uses a non-http scheme) to this localhost-only API.
func corsLocal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

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
// a CORS preflight that the server does not respond to.
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
	cfg, err := config.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}
	projects := cfg.Projects
	if projects == nil {
		projects = make(map[string]config.Project)
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

	var cfg config.Config
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
