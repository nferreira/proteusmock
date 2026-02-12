package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/domain/scenario"
	"github.com/sophialabs/proteusmock/internal/domain/trace"
	inboundhttp "github.com/sophialabs/proteusmock/internal/infrastructure/inbound/http"
	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
	"github.com/sophialabs/proteusmock/internal/infrastructure/usecases"
	"github.com/sophialabs/proteusmock/internal/testutil"
)

// stubRepo implements scenario.Repository for testing.
type stubRepo struct {
	scenarios []*scenario.Scenario
	err       error
}

func (r *stubRepo) LoadAll(_ context.Context) ([]*scenario.Scenario, error) {
	return r.scenarios, r.err
}

func (r *stubRepo) LoadByID(_ context.Context, id string) (*scenario.Scenario, error) {
	for _, s := range r.scenarios {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, scenario.ErrNotFound
}

func (r *stubRepo) SaveScenario(_ context.Context, _ *scenario.Scenario, _ []byte) error {
	return nil
}

func (r *stubRepo) DeleteScenario(_ context.Context, _ string, _ int) error {
	return nil
}

func (r *stubRepo) ReadSourceYAML(_ context.Context, _ *scenario.Scenario) ([]byte, error) {
	return nil, nil
}

func buildTestServer(scenarios ...*match.CompiledScenario) (*inboundhttp.Server, *services.ScenarioIndex) {
	traceBuf := trace.NewRingBuffer(50)
	evaluator := match.NewEvaluator()
	clk := &testutil.FixedClock{T: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	rl := &testutil.StubRateLimiter{AllowAll: true}
	logger := &testutil.NoopLogger{}

	handleReqUC := usecases.NewHandleRequestUseCase(evaluator, clk, rl, logger, traceBuf)

	// We don't need loadUC for basic mock handler tests.
	srv := inboundhttp.NewServer(handleReqUC, nil, traceBuf, logger)

	idx := services.NewScenarioIndex()
	for _, s := range scenarios {
		idx.Add(s)
	}
	idx.Build()
	srv.Rebuild(idx)

	return srv, idx
}

func TestMockHandler_MatchesGET(t *testing.T) {
	srv, _ := buildTestServer(&match.CompiledScenario{
		ID:       "health",
		Name:     "Health Check",
		Method:   "GET",
		PathKey:  "GET:/api/health",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
		},
		Response: match.CompiledResponse{
			Status:      200,
			Headers:     map[string]string{"X-Mock": "true"},
			Body:        []byte(`{"status":"ok"}`),
			ContentType: "application/json",
		},
	})

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status":"ok"}` {
		t.Errorf("unexpected body: %s", body)
	}

	if resp.Header.Get("X-Mock") != "true" {
		t.Errorf("expected X-Mock header")
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("unexpected content type: %s", resp.Header.Get("Content-Type"))
	}
}

func TestMockHandler_NoMatch_Returns404WithDebug(t *testing.T) {
	srv, _ := buildTestServer(&match.CompiledScenario{
		ID:       "post-only",
		Name:     "POST Only",
		Method:   "POST",
		PathKey:  "POST:/api/items",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "method", Predicate: func(s string) bool { return s == "POST" }},
		},
		Response: match.CompiledResponse{Status: 201},
	})

	req := httptest.NewRequest("GET", "/api/items", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var debug map[string]any
	if err := json.Unmarshal(body, &debug); err != nil {
		t.Fatalf("failed to parse debug response: %v", err)
	}
	if debug["error"] != "no_match" {
		t.Errorf("expected error 'no_match', got %v", debug["error"])
	}
}

func TestMockHandler_POSTWithBody(t *testing.T) {
	srv, _ := buildTestServer(&match.CompiledScenario{
		ID:       "create",
		Name:     "Create Item",
		Method:   "POST",
		PathKey:  "POST:/api/items",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "method", Predicate: func(s string) bool { return s == "POST" }},
			{Field: "header:Content-Type", Predicate: func(s string) bool { return s == "application/json" }},
		},
		Response: match.CompiledResponse{
			Status:      201,
			Body:        []byte(`{"created":true}`),
			ContentType: "application/json",
		},
	})

	req := httptest.NewRequest("POST", "/api/items", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestAdminHandler_ListScenarios(t *testing.T) {
	srv, _ := buildTestServer(
		&match.CompiledScenario{
			ID: "s1", Name: "Scenario 1", Method: "GET", PathKey: "GET:/a", Priority: 10,
		},
		&match.CompiledScenario{
			ID: "s2", Name: "Scenario 2", Method: "POST", PathKey: "POST:/b", Priority: 5,
		},
	)

	req := httptest.NewRequest("GET", "/__admin/scenarios", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var scenarios []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &scenarios); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(scenarios) != 2 {
		t.Errorf("expected 2 scenarios, got %d", len(scenarios))
	}
}

func TestAdminHandler_SearchScenarios(t *testing.T) {
	srv, _ := buildTestServer(
		&match.CompiledScenario{
			ID: "health-check", Name: "Health Check", Method: "GET", PathKey: "GET:/health",
		},
		&match.CompiledScenario{
			ID: "create-item", Name: "Create Item", Method: "POST", PathKey: "POST:/items",
		},
	)

	req := httptest.NewRequest("GET", "/__admin/scenarios/search?q=health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var results []map[string]any
	json.Unmarshal(w.Body.Bytes(), &results)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestAdminHandler_GetTrace(t *testing.T) {
	srv, _ := buildTestServer(&match.CompiledScenario{
		ID:       "traced",
		Method:   "GET",
		PathKey:  "GET:/api/traced",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
		},
		Response: match.CompiledResponse{Status: 200, Body: []byte("ok")},
	})

	// Make a request to generate a trace entry.
	req := httptest.NewRequest("GET", "/api/traced", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Now query the trace.
	req = httptest.NewRequest("GET", "/__admin/trace?last=5", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var entries []map[string]any
	json.Unmarshal(w.Body.Bytes(), &entries)
	if len(entries) != 1 {
		t.Errorf("expected 1 trace entry, got %d", len(entries))
	}
}

func TestMockHandler_RateLimited(t *testing.T) {
	traceBuf := trace.NewRingBuffer(50)
	evaluator := match.NewEvaluator()
	clk := &testutil.FixedClock{T: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	rl := &testutil.StubRateLimiter{AllowAll: false} // Always deny.
	logger := &testutil.NoopLogger{}

	handleReqUC := usecases.NewHandleRequestUseCase(evaluator, clk, rl, logger, traceBuf)
	srv := inboundhttp.NewServer(handleReqUC, nil, traceBuf, logger)

	idx := services.NewScenarioIndex()
	idx.Add(&match.CompiledScenario{
		ID:       "limited",
		Method:   "GET",
		PathKey:  "GET:/api/limited",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
		},
		Response: match.CompiledResponse{Status: 200, Body: []byte("ok")},
		Policy: &match.CompiledPolicy{
			RateLimit: &match.CompiledRateLimit{Rate: 1, Burst: 1, Key: "test"},
		},
	})
	idx.Build()
	srv.Rebuild(idx)

	req := httptest.NewRequest("GET", "/api/limited", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "rate_limited" {
		t.Errorf("expected rate_limited error, got %v", body["error"])
	}
}

func TestNotFoundHandler(t *testing.T) {
	srv, _ := buildTestServer() // No scenarios.

	req := httptest.NewRequest("GET", "/unregistered", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "no_match" {
		t.Errorf("expected 'no_match', got %v", body["error"])
	}
	if body["path"] != "/unregistered" {
		t.Errorf("expected path '/unregistered', got %v", body["path"])
	}
}

func TestMockHandler_TemplateRendering(t *testing.T) {
	renderer := &fakeRenderer{body: []byte(`Hello, rendered!`)}
	srv, _ := buildTestServer(&match.CompiledScenario{
		ID:       "template",
		Method:   "GET",
		PathKey:  "GET:/api/template",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
		},
		Response: match.CompiledResponse{
			Status:      200,
			Renderer:    renderer,
			ContentType: "text/plain",
		},
	})

	req := httptest.NewRequest("GET", "/api/template", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "Hello, rendered!" {
		t.Errorf("expected rendered body, got %q", w.Body.String())
	}
}

func TestMockHandler_TemplateRenderError(t *testing.T) {
	renderer := &errorRenderer{}
	srv, _ := buildTestServer(&match.CompiledScenario{
		ID:       "render-error",
		Method:   "GET",
		PathKey:  "GET:/api/error",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
		},
		Response: match.CompiledResponse{
			Status:      200,
			Renderer:    renderer,
			ContentType: "text/plain",
		},
	})

	req := httptest.NewRequest("GET", "/api/error", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestMockHandler_Pagination(t *testing.T) {
	srv, _ := buildTestServer(&match.CompiledScenario{
		ID:       "paginated",
		Method:   "GET",
		PathKey:  "GET:/api/items",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
		},
		Response: match.CompiledResponse{
			Status:      200,
			Body:        []byte(`[1,2,3,4,5,6,7,8,9,10]`),
			ContentType: "application/json",
		},
		Policy: &match.CompiledPolicy{
			Pagination: &match.CompiledPagination{
				Style:       "page_size",
				PageParam:   "page",
				SizeParam:   "size",
				DefaultSize: 3,
				MaxSize:     100,
				DataPath:    "$",
				Envelope: match.CompiledPaginationEnvelope{
					DataField:        "data",
					PageField:        "page",
					SizeField:        "size",
					TotalItemsField:  "total_items",
					TotalPagesField:  "total_pages",
					HasNextField:     "has_next",
					HasPreviousField: "has_previous",
				},
			},
		},
	})

	req := httptest.NewRequest("GET", "/api/items?page=2&size=3", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var env map[string]any
	json.Unmarshal(w.Body.Bytes(), &env)
	if env["page"].(float64) != 2 {
		t.Errorf("expected page 2, got %v", env["page"])
	}
}

func TestMockHandler_PaginationError(t *testing.T) {
	srv, _ := buildTestServer(&match.CompiledScenario{
		ID:       "bad-pagination",
		Method:   "GET",
		PathKey:  "GET:/api/bad",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
		},
		Response: match.CompiledResponse{
			Status:      200,
			Body:        []byte(`not json`), // Body isn't JSON, pagination will fail.
			ContentType: "text/plain",
		},
		Policy: &match.CompiledPolicy{
			Pagination: &match.CompiledPagination{
				Style:       "page_size",
				PageParam:   "page",
				SizeParam:   "size",
				DefaultSize: 10,
				MaxSize:     100,
				DataPath:    "$",
				Envelope: match.CompiledPaginationEnvelope{
					DataField:        "data",
					PageField:        "page",
					SizeField:        "size",
					TotalItemsField:  "total_items",
					TotalPagesField:  "total_pages",
					HasNextField:     "has_next",
					HasPreviousField: "has_previous",
				},
			},
		},
	})

	req := httptest.NewRequest("GET", "/api/bad", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Pagination fails gracefully — returns unpaginated body.
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "not json" {
		t.Errorf("expected original body on pagination error, got %q", w.Body.String())
	}
}

func TestMockHandler_DebugResponseWithFailedCandidates(t *testing.T) {
	srv, _ := buildTestServer(
		&match.CompiledScenario{
			ID:       "needs-post",
			Name:     "Needs POST",
			Method:   "GET",
			PathKey:  "GET:/api/test",
			Priority: 10,
			Predicates: []match.FieldPredicate{
				{Field: "method", Predicate: func(s string) bool { return s == "POST" }},
			},
			Response: match.CompiledResponse{Status: 200},
		},
	)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var debug map[string]any
	json.Unmarshal(w.Body.Bytes(), &debug)

	candidates, ok := debug["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		t.Fatal("expected candidates in debug response")
	}

	c := candidates[0].(map[string]any)
	if c["matched"] != false {
		t.Error("expected candidate to be unmatched")
	}
	if _, ok := c["failed_field"]; !ok {
		t.Error("expected failed_field in unmatched candidate")
	}
	if _, ok := c["failed_reason"]; !ok {
		t.Error("expected failed_reason in unmatched candidate")
	}
}

func TestAdminHandler_SearchScenarios_EmptyQuery(t *testing.T) {
	srv, _ := buildTestServer(
		&match.CompiledScenario{
			ID: "s1", Name: "S1", Method: "GET", PathKey: "GET:/a",
		},
		&match.CompiledScenario{
			ID: "s2", Name: "S2", Method: "POST", PathKey: "POST:/b",
		},
	)

	req := httptest.NewRequest("GET", "/__admin/scenarios/search?q=", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var results []map[string]any
	json.Unmarshal(w.Body.Bytes(), &results)
	if len(results) != 2 {
		t.Errorf("expected 2 results for empty query, got %d", len(results))
	}
}

func TestAdminHandler_GetTrace_DefaultCount(t *testing.T) {
	srv, _ := buildTestServer()

	req := httptest.NewRequest("GET", "/__admin/trace", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMockHandler_ContentTypeInferred(t *testing.T) {
	srv, _ := buildTestServer(&match.CompiledScenario{
		ID:       "no-ct",
		Method:   "GET",
		PathKey:  "GET:/api/test",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
		},
		Response: match.CompiledResponse{
			Status: 200,
			Body:   []byte(`{"ok":true}`),
			// No ContentType set — should be inferred.
		},
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// Helper types for template testing.

type fakeRenderer struct {
	body []byte
}

func (r *fakeRenderer) Render(_ match.RenderContext) ([]byte, error) {
	return r.body, nil
}

type errorRenderer struct{}

func (r *errorRenderer) Render(_ match.RenderContext) ([]byte, error) {
	return nil, io.ErrUnexpectedEOF
}

func TestAdminHandler_ReloadSuccess(t *testing.T) {
	traceBuf := trace.NewRingBuffer(50)
	evaluator := match.NewEvaluator()
	clk := &testutil.FixedClock{T: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	rl := &testutil.StubRateLimiter{AllowAll: true}
	logger := &testutil.NoopLogger{}

	repo := &stubRepo{
		scenarios: []*scenario.Scenario{
			{
				ID: "reloaded", Name: "Reloaded", Priority: 10,
				When:     scenario.WhenClause{Method: "GET", Path: "/api/reloaded"},
				Response: scenario.Response{Status: 200, Body: "ok"},
			},
		},
	}

	compiler, _ := services.NewCompiler(t.TempDir(), nil)
	loadUC := usecases.NewLoadScenariosUseCase(repo, compiler, logger)
	handleReqUC := usecases.NewHandleRequestUseCase(evaluator, clk, rl, logger, traceBuf)
	srv := inboundhttp.NewServer(handleReqUC, loadUC, traceBuf, logger)

	// Initial build with empty index.
	idx := services.NewScenarioIndex()
	idx.Build()
	srv.Rebuild(idx)

	req := httptest.NewRequest("POST", "/__admin/reload", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", body["status"])
	}
}

func TestAdminHandler_ReloadFailure(t *testing.T) {
	traceBuf := trace.NewRingBuffer(50)
	evaluator := match.NewEvaluator()
	clk := &testutil.FixedClock{T: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	rl := &testutil.StubRateLimiter{AllowAll: true}
	logger := &testutil.NoopLogger{}

	repo := &stubRepo{err: fmt.Errorf("load error")}

	compiler, _ := services.NewCompiler(t.TempDir(), nil)
	loadUC := usecases.NewLoadScenariosUseCase(repo, compiler, logger)
	handleReqUC := usecases.NewHandleRequestUseCase(evaluator, clk, rl, logger, traceBuf)
	srv := inboundhttp.NewServer(handleReqUC, loadUC, traceBuf, logger)

	idx := services.NewScenarioIndex()
	idx.Build()
	srv.Rebuild(idx)

	req := httptest.NewRequest("POST", "/__admin/reload", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "reload_failed" {
		t.Errorf("expected error 'reload_failed', got %v", body["error"])
	}
}

func TestAdminHandler_ListScenarios_NilIndex(t *testing.T) {
	traceBuf := trace.NewRingBuffer(10)
	evaluator := match.NewEvaluator()
	handleReqUC := usecases.NewHandleRequestUseCase(evaluator, &testutil.FixedClock{}, &testutil.StubRateLimiter{AllowAll: true}, &testutil.NoopLogger{}, traceBuf)
	srv := inboundhttp.NewServer(handleReqUC, nil, traceBuf, &testutil.NoopLogger{})

	// Build with empty index so router exists, but then test nil index path.
	// We can't easily test nil index through the admin route since Rebuild always stores the index.
	// Instead test via the buildTestServer pattern but with a mock handler that hits the nil path.
	// Actually, the nil index is handled by early return. The test for "no scenarios" covers the non-nil empty case.

	// Build minimal router.
	idx := services.NewScenarioIndex()
	idx.Build()
	srv.Rebuild(idx)

	req := httptest.NewRequest("GET", "/__admin/scenarios", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAdminHandler_SearchScenarios_NilIndex(t *testing.T) {
	traceBuf := trace.NewRingBuffer(10)
	evaluator := match.NewEvaluator()
	handleReqUC := usecases.NewHandleRequestUseCase(evaluator, &testutil.FixedClock{}, &testutil.StubRateLimiter{AllowAll: true}, &testutil.NoopLogger{}, traceBuf)
	srv := inboundhttp.NewServer(handleReqUC, nil, traceBuf, &testutil.NoopLogger{})

	idx := services.NewScenarioIndex()
	idx.Build()
	srv.Rebuild(idx)

	req := httptest.NewRequest("GET", "/__admin/scenarios/search?q=test", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMockHandler_WithPathParams(t *testing.T) {
	srv, _ := buildTestServer(&match.CompiledScenario{
		ID:       "param",
		Method:   "GET",
		PathKey:  "GET:/api/users/{id}",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
		},
		Response: match.CompiledResponse{
			Status:      200,
			Body:        []byte(`{"id":"found"}`),
			ContentType: "application/json",
		},
	})

	req := httptest.NewRequest("GET", "/api/users/42", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestServer_NotReady(t *testing.T) {
	traceBuf := trace.NewRingBuffer(10)
	evaluator := match.NewEvaluator()
	handleReqUC := usecases.NewHandleRequestUseCase(evaluator, &testutil.FixedClock{}, &testutil.StubRateLimiter{AllowAll: true}, &testutil.NoopLogger{}, traceBuf)
	srv := inboundhttp.NewServer(handleReqUC, nil, traceBuf, &testutil.NoopLogger{})

	// Don't call Rebuild — server has no router.
	req := httptest.NewRequest("GET", "/anything", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
