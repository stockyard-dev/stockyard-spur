package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/stockyard-dev/stockyard-spur/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	port   int
	limits Limits
}

func New(db *store.DB, port int, limits Limits) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), port: port, limits: limits}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Projects
	s.mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	s.mux.HandleFunc("GET /api/projects", s.handleListProjects)
	s.mux.HandleFunc("GET /api/projects/{id}", s.handleGetProject)
	s.mux.HandleFunc("DELETE /api/projects/{id}", s.handleDeleteProject)

	// Endpoints
	s.mux.HandleFunc("POST /api/projects/{id}/endpoints", s.handleCreateEndpoint)
	s.mux.HandleFunc("GET /api/projects/{id}/endpoints", s.handleListEndpoints)
	s.mux.HandleFunc("GET /api/endpoints/{id}", s.handleGetEndpoint)
	s.mux.HandleFunc("PUT /api/endpoints/{id}", s.handleUpdateEndpoint)
	s.mux.HandleFunc("DELETE /api/endpoints/{id}", s.handleDeleteEndpoint)

	// Request log
	s.mux.HandleFunc("GET /api/endpoints/{id}/log", s.handleRequestLog)

	// Status
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /ui", s.handleUI)
	s.mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"product": "stockyard-spur", "version": "0.1.0"})
	})

	// Mock catch-all under /mock/ — serves fake responses
	s.mux.HandleFunc("/mock/", s.handleMock)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[spur] listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

// --- Mock handler (the hot path) ---

func (s *Server) handleMock(w http.ResponseWriter, r *http.Request) {
	// Strip /mock prefix to get the endpoint path
	path := strings.TrimPrefix(r.URL.Path, "/mock")
	if path == "" {
		path = "/"
	}

	ep, err := s.db.MatchEndpoint(r.Method, path)
	if err != nil {
		// Try wildcard GET match for any method
		ep, err = s.db.MatchEndpoint("*", path)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "no mock endpoint matching " + r.Method + " " + path})
			return
		}
	}

	// Log the request
	if s.limits.RequestLog {
		hdrs := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				hdrs[k] = v[0]
			}
		}
		headersJSON, _ := json.Marshal(hdrs)
		body, _ := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = strings.Split(fwd, ",")[0]
		}
		go s.db.LogRequest(ep.ID, r.Method, path, string(headersJSON), string(body), ip)
	}

	// Simulated delay (Pro)
	if ep.DelayMs > 0 && s.limits.DelaySupport {
		time.Sleep(time.Duration(ep.DelayMs) * time.Millisecond)
	}

	// Set response headers
	var respHeaders map[string]string
	if err := json.Unmarshal([]byte(ep.ResponseHeaders), &respHeaders); err == nil {
		for k, v := range respHeaders {
			w.Header().Set(k, v)
		}
	}

	w.Header().Set("X-Spur-Endpoint", ep.ID)
	w.WriteHeader(ep.StatusCode)
	w.Write([]byte(ep.ResponseBody))
}

// --- Project handlers ---

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		BasePath string `json:"base_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, 400, map[string]string{"error": "name is required"})
		return
	}
	if s.limits.MaxProjects > 0 {
		projects, _ := s.db.ListProjects()
		if LimitReached(s.limits.MaxProjects, len(projects)) {
			writeJSON(w, 402, map[string]string{"error": fmt.Sprintf("free tier limit: %d projects — upgrade to Pro", s.limits.MaxProjects), "upgrade": "https://stockyard.dev/spur/"})
			return
		}
	}
	p, err := s.db.CreateProject(req.Name, req.BasePath)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]any{"project": p})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.db.ListProjects()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if projects == nil {
		projects = []store.Project{}
	}
	writeJSON(w, 200, map[string]any{"projects": projects, "count": len(projects)})
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	p, err := s.db.GetProject(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "project not found"})
		return
	}
	writeJSON(w, 200, map[string]any{"project": p})
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteProject(r.PathValue("id"))
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// --- Endpoint handlers ---

func (s *Server) handleCreateEndpoint(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, err := s.db.GetProject(projectID); err != nil {
		writeJSON(w, 404, map[string]string{"error": "project not found"})
		return
	}

	var req struct {
		Method          string `json:"method"`
		Path            string `json:"path"`
		StatusCode      int    `json:"status_code"`
		ResponseBody    string `json:"response_body"`
		ResponseHeaders string `json:"response_headers"`
		DelayMs         int    `json:"delay_ms"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Path == "" {
		writeJSON(w, 400, map[string]string{"error": "path is required"})
		return
	}

	if s.limits.MaxEndpoints > 0 {
		total := s.db.TotalEndpoints()
		if LimitReached(s.limits.MaxEndpoints, total) {
			writeJSON(w, 402, map[string]string{"error": fmt.Sprintf("free tier limit: %d endpoints — upgrade to Pro", s.limits.MaxEndpoints), "upgrade": "https://stockyard.dev/spur/"})
			return
		}
	}

	if req.DelayMs > 0 && !s.limits.DelaySupport {
		req.DelayMs = 0 // silently ignore on free tier
	}

	// Marshal response_headers if passed as object
	respHeaders := req.ResponseHeaders
	if respHeaders == "" {
		respHeaders = `{"Content-Type":"application/json"}`
	}

	ep, err := s.db.CreateEndpoint(projectID, req.Method, req.Path, req.StatusCode, req.ResponseBody, respHeaders, req.DelayMs)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	mockURL := fmt.Sprintf("http://localhost:%d/mock%s", s.port, ep.Path)
	writeJSON(w, 201, map[string]any{"endpoint": ep, "mock_url": mockURL})
}

func (s *Server) handleListEndpoints(w http.ResponseWriter, r *http.Request) {
	eps, err := s.db.ListEndpoints(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if eps == nil {
		eps = []store.Endpoint{}
	}
	writeJSON(w, 200, map[string]any{"endpoints": eps, "count": len(eps)})
}

func (s *Server) handleGetEndpoint(w http.ResponseWriter, r *http.Request) {
	ep, err := s.db.GetEndpoint(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "endpoint not found"})
		return
	}
	writeJSON(w, 200, map[string]any{"endpoint": ep})
}

func (s *Server) handleUpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.db.GetEndpoint(id); err != nil {
		writeJSON(w, 404, map[string]string{"error": "endpoint not found"})
		return
	}
	var req struct {
		StatusCode      *int    `json:"status_code"`
		ResponseBody    *string `json:"response_body"`
		ResponseHeaders *string `json:"response_headers"`
		DelayMs         *int    `json:"delay_ms"`
		Enabled         *bool   `json:"enabled"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	ep, err := s.db.UpdateEndpoint(id, req.StatusCode, req.ResponseBody, req.ResponseHeaders, req.DelayMs, req.Enabled)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"endpoint": ep})
}

func (s *Server) handleDeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteEndpoint(r.PathValue("id"))
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) handleRequestLog(w http.ResponseWriter, r *http.Request) {
	entries, err := s.db.ListRequestLog(r.PathValue("id"), 50)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []store.RequestEntry{}
	}
	writeJSON(w, 200, map[string]any{"requests": entries, "count": len(entries)})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.db.Stats())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
