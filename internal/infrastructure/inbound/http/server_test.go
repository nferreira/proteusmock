package http_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/domain/trace"
	inboundhttp "github.com/sophialabs/proteusmock/internal/infrastructure/inbound/http"
	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
	"github.com/sophialabs/proteusmock/internal/infrastructure/usecases"
	"github.com/sophialabs/proteusmock/internal/testutil"
)

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
