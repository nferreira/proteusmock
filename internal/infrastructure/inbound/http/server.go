package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/domain/trace"
	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
	"github.com/sophialabs/proteusmock/internal/infrastructure/usecases"
)

const maxBodySize = 10 << 20 // 10 MB

// Server is the main HTTP server for ProteusMock.
type Server struct {
	router      atomic.Pointer[chi.Mux]
	index       atomic.Pointer[services.ScenarioIndex]
	rebuildMu   sync.Mutex
	handleReqUC *usecases.HandleRequestUseCase
	loadUC      *usecases.LoadScenariosUseCase
	traceBuf    *trace.RingBuffer
	logger      ports.Logger
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

// BuildRouter creates a new chi.Mux with admin and mock routes for the given index.
func (s *Server) BuildRouter(idx *services.ScenarioIndex) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Admin routes.
	r.Route("/__admin", func(r chi.Router) {
		r.Get("/scenarios", s.handleListScenarios)
		r.Get("/scenarios/search", s.handleSearchScenarios)
		r.Get("/trace", s.handleGetTrace)
		r.Post("/reload", s.handleReload)
	})

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

func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
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
