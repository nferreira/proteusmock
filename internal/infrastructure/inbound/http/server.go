package http

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/domain/scenario"
	"github.com/sophialabs/proteusmock/internal/domain/trace"
	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
	"github.com/sophialabs/proteusmock/internal/infrastructure/usecases"
	dashboard "github.com/sophialabs/proteusmock/ui/dashboard"
)

const maxBodySize = 10 << 20 // 10 MB

// Server is the main HTTP server for ProteusMock.
type Server struct {
	router      atomic.Pointer[chi.Mux]
	index       atomic.Pointer[services.ScenarioIndex]
	rebuildMu   sync.Mutex
	handleReqUC *usecases.HandleRequestUseCase
	loadUC      *usecases.LoadScenariosUseCase
	saveUC      *usecases.SaveScenarioUseCase
	deleteUC    *usecases.DeleteScenarioUseCase
	repo        scenario.Repository
	traceBuf    *trace.RingBuffer
	logger      ports.Logger
	rootDir     string
}

// NewServer creates a new Server.
func NewServer(
	handleReqUC *usecases.HandleRequestUseCase,
	loadUC *usecases.LoadScenariosUseCase,
	traceBuf *trace.RingBuffer,
	logger ports.Logger,
) *Server {
	s := &Server{
		handleReqUC: handleReqUC,
		loadUC:      loadUC,
		traceBuf:    traceBuf,
		logger:      logger,
	}
	return s
}

// SetCRUDDeps injects the optional CRUD dependencies (save, delete use cases and repo).
// This is separated from NewServer to maintain backward compatibility with existing callers.
func (s *Server) SetCRUDDeps(saveUC *usecases.SaveScenarioUseCase, deleteUC *usecases.DeleteScenarioUseCase, repo scenario.Repository, rootDir string) {
	s.saveUC = saveUC
	s.deleteUC = deleteUC
	s.repo = repo
	s.rootDir = rootDir
}

// BuildRouter creates a new chi.Mux with admin and mock routes for the given index.
func (s *Server) BuildRouter(idx *services.ScenarioIndex) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Admin routes.
	r.Route("/__admin", func(r chi.Router) {
		r.Get("/scenarios", s.handleListScenarios)
		r.Get("/scenarios/search", s.handleSearchScenarios)
		r.Get("/scenarios/{scenarioID}", s.handleGetScenario)
		r.Put("/scenarios/{scenarioID}", s.handleUpdateScenario)
		r.Post("/scenarios", s.handleCreateScenario)
		r.Delete("/scenarios/{scenarioID}", s.handleDeleteScenario)
		r.Get("/files", s.handleListFiles)
		r.Get("/trace", s.handleGetTrace)
		r.Post("/reload", s.handleReload)
	})

	// Dashboard SPA (embedded). Serves files directly to avoid http.FileServer redirect loops.
	dist, _ := fs.Sub(dashboard.DistFS, "dist")
	serveDashboard := s.dashboardHandler(dist)
	r.Get("/__ui", serveDashboard)
	r.Get("/__ui/*", serveDashboard)

	// Dynamic mock routes from index.
	for _, path := range idx.Paths() {
		routePath := path
		r.HandleFunc(routePath, s.mockHandler)
	}

	// Catch-all for unmatched paths — returns 404 with debug info.
	r.NotFound(s.notFoundHandler)

	return r
}

// Rebuild atomically swaps the router and index. Serialized via mutex.
func (s *Server) Rebuild(idx *services.ScenarioIndex) {
	s.rebuildMu.Lock()
	defer s.rebuildMu.Unlock()

	r := s.BuildRouter(idx)
	s.index.Store(idx)
	s.router.Store(r)
	s.logger.Info("router rebuilt", "paths", len(idx.Paths()))
}

// ServeHTTP implements http.Handler using the atomic router.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router := s.router.Load()
	if router == nil {
		http.Error(w, "server not ready", http.StatusServiceUnavailable)
		return
	}
	router.ServeHTTP(w, r)
}

// dashboardHandler returns an http.HandlerFunc that serves the embedded SPA files.
func (s *Server) dashboardHandler(dist fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Strip the prefix to get the file path within dist/.
		filePath := strings.TrimPrefix(r.URL.Path, "/__ui/")
		if filePath == "" || filePath == "__ui" {
			filePath = "index.html"
		}

		// Try to open the requested file; fall back to index.html for SPA client-side routing.
		f, err := dist.Open(filePath)
		if err != nil {
			filePath = "index.html"
			f, err = dist.Open(filePath)
			if err != nil {
				http.Error(w, "dashboard not available", http.StatusNotFound)
				return
			}
		}
		defer f.Close()

		// If it's a directory (e.g. /assets/), serve index.html instead.
		if info, _ := f.Stat(); info != nil && info.IsDir() {
			f.Close()
			filePath = "index.html"
			f, err = dist.Open(filePath)
			if err != nil {
				http.Error(w, "dashboard not available", http.StatusNotFound)
				return
			}
			defer f.Close()
		}

		// Detect content type from extension.
		contentType := "application/octet-stream"
		switch {
		case strings.HasSuffix(filePath, ".html"):
			contentType = "text/html; charset=utf-8"
		case strings.HasSuffix(filePath, ".css"):
			contentType = "text/css; charset=utf-8"
		case strings.HasSuffix(filePath, ".js"):
			contentType = "application/javascript; charset=utf-8"
		case strings.HasSuffix(filePath, ".json"):
			contentType = "application/json"
		case strings.HasSuffix(filePath, ".svg"):
			contentType = "image/svg+xml"
		case strings.HasSuffix(filePath, ".png"):
			contentType = "image/png"
		case strings.HasSuffix(filePath, ".ico"):
			contentType = "image/x-icon"
		case strings.HasSuffix(filePath, ".woff2"):
			contentType = "font/woff2"
		case strings.HasSuffix(filePath, ".woff"):
			contentType = "font/woff"
		}

		w.Header().Set("Content-Type", contentType)
		data, _ := io.ReadAll(f.(io.Reader))
		w.Write(data)
	}
}

func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("request received (no route)", "method", r.Method, "path", r.URL.Path, "query", r.URL.RawQuery, "remote", r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	writeJSON(w, map[string]any{
		"error":   "no_match",
		"method":  r.Method,
		"path":    r.URL.Path,
		"message": "No scenario registered for this path",
	})
}

func (s *Server) mockHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("request received", "method", r.Method, "path", r.URL.Path, "query", r.URL.RawQuery, "remote", r.RemoteAddr)

	defer func() { _ = r.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// Canonicalize header keys to http.CanonicalHeaderKey for consistent matching.
	headers := make(map[string]string)
	for k := range r.Header {
		headers[http.CanonicalHeaderKey(k)] = r.Header.Get(k)
	}

	incoming := &match.IncomingRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: headers,
		Body:    body,
	}

	idx := s.index.Load()
	if idx == nil {
		http.Error(w, "server not ready", http.StatusServiceUnavailable)
		return
	}

	// Use the chi route pattern (e.g. /api/v1/echo/{id}) for index lookup,
	// falling back to the actual path if no pattern is available.
	routePath := r.URL.Path
	if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
		routePath = rctx.RoutePattern()
	}
	key := r.Method + ":" + routePath
	candidates := idx.Lookup(key)

	result := s.handleReqUC.Execute(r.Context(), incoming, candidates)

	if result.RateLimited {
		s.logger.Info("request rate-limited", "method", r.Method, "path", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		writeJSON(w, map[string]string{
			"error":   "rate_limited",
			"message": "Too many requests",
		})
		return
	}

	if !result.Matched {
		s.logger.Info("request unmatched", "method", r.Method, "path", r.URL.Path, "candidates", len(result.TraceEntry.Candidates))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		debugResp := buildDebugResponse(r.Method, r.URL.Path, result.TraceEntry)
		writeJSON(w, debugResp)
		return
	}

	resp := result.Response

	// Render dynamic body if template renderer is present.
	queryParams := extractQueryParams(r)
	var bodyBytes []byte
	if resp.Renderer != nil {
		renderCtx := match.RenderContext{
			Method:      r.Method,
			Path:        r.URL.Path,
			Headers:     headers,
			QueryParams: queryParams,
			PathParams:  extractPathParams(r),
			Body:        body,
			Now:         time.Now().UTC().Format(time.RFC3339),
		}
		rendered, renderErr := resp.Renderer.Render(renderCtx)
		if renderErr != nil {
			s.logger.Error("template render failed", "error", renderErr)
			http.Error(w, "template render error", http.StatusInternalServerError)
			return
		}
		bodyBytes = rendered
	} else {
		bodyBytes = resp.Body
	}

	// Pagination post-processing: slice the rendered body and wrap in envelope.
	if result.Pagination != nil {
		paginated, paginateErr := services.Paginate(bodyBytes, result.Pagination, queryParams)
		if paginateErr != nil {
			s.logger.Error("pagination failed, returning unpaginated response", "error", paginateErr)
		} else {
			bodyBytes = paginated
		}
	}

	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	if resp.ContentType != "" {
		w.Header().Set("Content-Type", resp.ContentType)
	}
	w.WriteHeader(resp.Status)
	if _, err := w.Write(bodyBytes); err != nil {
		s.logger.Debug("failed to write response body", "error", err)
	}

	s.logger.Info("request matched", "method", r.Method, "path", r.URL.Path, "scenario", result.TraceEntry.MatchedID, "status", resp.Status)
}

func buildDebugResponse(method, path string, entry trace.Entry) map[string]any {
	resp := map[string]any{
		"error":   "no_match",
		"method":  method,
		"path":    path,
		"message": "No scenario matched the request",
	}

	if len(entry.Candidates) > 0 {
		candidates := make([]map[string]any, 0, len(entry.Candidates))
		for _, c := range entry.Candidates {
			cm := map[string]any{
				"scenario_id":   c.ScenarioID,
				"scenario_name": c.ScenarioName,
				"matched":       c.Matched,
			}
			if !c.Matched {
				cm["failed_field"] = c.FailedField
				cm["failed_reason"] = c.FailedReason
			}
			candidates = append(candidates, cm)
		}
		resp["candidates"] = candidates
	}

	return resp
}

func (s *Server) handleListScenarios(w http.ResponseWriter, _ *http.Request) {
	idx := s.index.Load()
	if idx == nil {
		writeJSON(w, []any{})
		return
	}

	all := idx.All()
	scenarios := make([]map[string]any, 0, len(all))
	for _, cs := range all {
		scenarios = append(scenarios, map[string]any{
			"id":       cs.ID,
			"name":     cs.Name,
			"priority": cs.Priority,
			"method":   cs.Method,
			"path_key": cs.PathKey,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, scenarios)
}

func (s *Server) handleSearchScenarios(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	idx := s.index.Load()
	if idx == nil {
		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, []any{})
		return
	}

	var results []map[string]any
	for _, cs := range idx.All() {
		if q == "" ||
			strings.Contains(strings.ToLower(cs.ID), q) ||
			strings.Contains(strings.ToLower(cs.Name), q) ||
			strings.Contains(strings.ToLower(cs.PathKey), q) {
			results = append(results, map[string]any{
				"id":       cs.ID,
				"name":     cs.Name,
				"priority": cs.Priority,
				"method":   cs.Method,
				"path_key": cs.PathKey,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, results)
}

func (s *Server) handleListFiles(w http.ResponseWriter, _ *http.Request) {
	if s.rootDir == "" {
		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, []string{})
		return
	}

	var files []string
	err := filepath.WalkDir(s.rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(s.rootDir, path)
		if relErr != nil {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		s.logger.Error("failed to list files", "error", err)
	}

	if files == nil {
		files = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, files)
}

func (s *Server) handleGetTrace(w http.ResponseWriter, r *http.Request) {
	n := 10
	if lastParam := r.URL.Query().Get("last"); lastParam != "" {
		if parsed, err := strconv.Atoi(lastParam); err == nil && parsed > 0 {
			n = parsed
		}
	}

	entries := s.traceBuf.Last(n)
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, entries)
}

func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	idx, err := s.loadUC.Execute(r.Context())
	if err != nil {
		s.logger.Error("reload failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{
			"error":   "reload_failed",
			"message": "scenario reload failed, check server logs",
		})
		return
	}

	s.Rebuild(idx)
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]string{
		"status":  "ok",
		"message": "scenarios reloaded",
	})
}

func (s *Server) handleGetScenario(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "scenarioID")
	if s.repo == nil {
		http.Error(w, "CRUD operations not configured", http.StatusNotImplemented)
		return
	}

	sc, err := s.repo.LoadByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, scenario.ErrNotFound) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"error": "not_found", "message": "scenario not found: " + id})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "internal", "message": err.Error()})
		return
	}

	// Read the raw YAML source.
	sourceYAML, err := s.repo.ReadSourceYAML(r.Context(), sc)
	if err != nil {
		s.logger.Warn("failed to read source YAML", "id", id, "error", err)
	}

	// Build relative source path for display.
	relPath := sc.SourceFile
	if s.rootDir != "" {
		if rel, err := filepath.Rel(s.rootDir, sc.SourceFile); err == nil {
			relPath = rel
		}
	}

	resp := map[string]any{
		"id":           sc.ID,
		"name":         sc.Name,
		"priority":     sc.Priority,
		"source_file":  relPath,
		"source_index": sc.SourceIndex,
		"source_yaml":  string(sourceYAML),
		"when":         buildWhenJSON(sc),
		"response":     buildResponseJSON(sc),
	}
	if sc.Policy != nil {
		resp["policy"] = buildPolicyJSON(sc.Policy)
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, resp)
}

func (s *Server) handleUpdateScenario(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "scenarioID")
	if s.saveUC == nil {
		http.Error(w, "CRUD operations not configured", http.StatusNotImplemented)
		return
	}

	defer func() { _ = r.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	if err := s.saveUC.Execute(r.Context(), id, body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "save_failed", "message": err.Error()})
		return
	}

	// Reload and rebuild.
	idx, err := s.loadUC.Execute(r.Context())
	if err != nil {
		s.logger.Error("reload after save failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "reload_failed", "message": err.Error()})
		return
	}
	s.Rebuild(idx)

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]string{"status": "ok", "message": "scenario updated", "id": id})
}

func (s *Server) handleCreateScenario(w http.ResponseWriter, r *http.Request) {
	if s.saveUC == nil {
		http.Error(w, "CRUD operations not configured", http.StatusNotImplemented)
		return
	}

	defer func() { _ = r.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	if err := s.saveUC.Execute(r.Context(), "", body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "create_failed", "message": err.Error()})
		return
	}

	// Reload and rebuild.
	idx, err := s.loadUC.Execute(r.Context())
	if err != nil {
		s.logger.Error("reload after create failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "reload_failed", "message": err.Error()})
		return
	}
	s.Rebuild(idx)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]string{"status": "ok", "message": "scenario created"})
}

func (s *Server) handleDeleteScenario(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "scenarioID")
	if s.deleteUC == nil {
		http.Error(w, "CRUD operations not configured", http.StatusNotImplemented)
		return
	}

	if err := s.deleteUC.Execute(r.Context(), id); err != nil {
		if errors.Is(err, scenario.ErrNotFound) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"error": "not_found", "message": "scenario not found: " + id})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "delete_failed", "message": err.Error()})
		return
	}

	// Reload and rebuild.
	idx, err := s.loadUC.Execute(r.Context())
	if err != nil {
		s.logger.Error("reload after delete failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "reload_failed", "message": err.Error()})
		return
	}
	s.Rebuild(idx)

	w.WriteHeader(http.StatusNoContent)
}

// JSON builders for scenario detail response.

func buildWhenJSON(sc *scenario.Scenario) map[string]any {
	when := map[string]any{
		"method": sc.When.Method,
		"path":   sc.When.Path,
	}
	if len(sc.When.Headers) > 0 {
		headers := make(map[string]string, len(sc.When.Headers))
		for k, v := range sc.When.Headers {
			headers[k] = v.Value()
		}
		when["headers"] = headers
	}
	if sc.When.Body != nil {
		when["body"] = buildBodyClauseJSON(sc.When.Body)
	}
	return when
}

func buildBodyClauseJSON(bc *scenario.BodyClause) map[string]any {
	result := map[string]any{}
	if bc.ContentType != "" {
		result["content_type"] = bc.ContentType
	}
	if len(bc.Conditions) > 0 {
		conds := make([]map[string]string, 0, len(bc.Conditions))
		for _, c := range bc.Conditions {
			conds = append(conds, map[string]string{
				"extractor": c.Extractor,
				"matcher":   c.Matcher.Value(),
			})
		}
		result["conditions"] = conds
	}
	if len(bc.All) > 0 {
		all := make([]map[string]any, 0, len(bc.All))
		for i := range bc.All {
			all = append(all, buildBodyClauseJSON(&bc.All[i]))
		}
		result["all"] = all
	}
	if len(bc.Any) > 0 {
		any := make([]map[string]any, 0, len(bc.Any))
		for i := range bc.Any {
			any = append(any, buildBodyClauseJSON(&bc.Any[i]))
		}
		result["any"] = any
	}
	if bc.Not != nil {
		result["not"] = buildBodyClauseJSON(bc.Not)
	}
	return result
}

func buildResponseJSON(sc *scenario.Scenario) map[string]any {
	resp := map[string]any{
		"status": sc.Response.Status,
	}
	if len(sc.Response.Headers) > 0 {
		resp["headers"] = sc.Response.Headers
	}
	if sc.Response.Body != "" {
		resp["body"] = sc.Response.Body
	}
	if sc.Response.BodyFile != "" {
		resp["body_file"] = sc.Response.BodyFile
	}
	if sc.Response.ContentType != "" {
		resp["content_type"] = sc.Response.ContentType
	}
	if sc.Response.Engine != "" {
		resp["engine"] = sc.Response.Engine
	}
	return resp
}

func buildPolicyJSON(p *scenario.Policy) map[string]any {
	result := map[string]any{}
	if p.RateLimit != nil {
		result["rate_limit"] = map[string]any{
			"rate":  p.RateLimit.Rate,
			"burst": p.RateLimit.Burst,
			"key":   p.RateLimit.Key,
		}
	}
	if p.Latency != nil {
		result["latency"] = map[string]any{
			"fixed_ms":  p.Latency.FixedMs,
			"jitter_ms": p.Latency.JitterMs,
		}
	}
	if p.Pagination != nil {
		pg := map[string]any{
			"style":        string(p.Pagination.Style),
			"default_size": p.Pagination.DefaultSize,
			"max_size":     p.Pagination.MaxSize,
			"data_path":    p.Pagination.DataPath,
		}
		result["pagination"] = pg
	}
	return result
}

func extractQueryParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	return params
}

func extractPathParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	rctx := chi.RouteContext(r.Context())
	if rctx != nil {
		for i, key := range rctx.URLParams.Keys {
			if i < len(rctx.URLParams.Values) {
				params[key] = rctx.URLParams.Values[i]
			}
		}
	}
	return params
}

func writeJSON(w http.ResponseWriter, v any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
