package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/domain/trace"
	"github.com/sophialabs/proteusmock/internal/infrastructure/usecases"
	"github.com/sophialabs/proteusmock/internal/testutil"
)

func newHandleRequestUC(allow bool) *usecases.HandleRequestUseCase {
	return usecases.NewHandleRequestUseCase(
		match.NewEvaluator(),
		&testutil.FixedClock{T: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
		&testutil.StubRateLimiter{AllowAll: allow},
		&testutil.NoopLogger{},
		trace.NewRingBuffer(50),
	)
}

func TestHandleRequest_NoMatch(t *testing.T) {
	uc := newHandleRequestUC(true)
	req := &match.IncomingRequest{
		Method: "GET",
		Path:   "/nonexistent",
	}
	result := uc.Execute(context.Background(), req, nil)

	if result.Matched {
		t.Error("expected no match")
	}
	if result.RateLimited {
		t.Error("expected not rate limited")
	}
}

func TestHandleRequest_Match(t *testing.T) {
	uc := newHandleRequestUC(true)
	req := &match.IncomingRequest{
		Method:  "GET",
		Path:    "/api/health",
		Headers: map[string]string{},
	}
	candidates := []*match.CompiledScenario{
		{
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
				Body:        []byte(`{"status":"ok"}`),
				ContentType: "application/json",
			},
		},
	}

	result := uc.Execute(context.Background(), req, candidates)

	if !result.Matched {
		t.Fatal("expected match")
	}
	if result.Response == nil {
		t.Fatal("expected response")
	}
	if result.Response.Status != 200 {
		t.Errorf("expected status 200, got %d", result.Response.Status)
	}
	if result.Response.ContentType != "application/json" {
		t.Errorf("expected application/json, got %s", result.Response.ContentType)
	}
}

func TestHandleRequest_RateLimited(t *testing.T) {
	uc := newHandleRequestUC(false) // Always deny.
	req := &match.IncomingRequest{
		Method:  "GET",
		Path:    "/api/limited",
		Headers: map[string]string{},
	}
	candidates := []*match.CompiledScenario{
		{
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
		},
	}

	result := uc.Execute(context.Background(), req, candidates)

	if !result.Matched {
		t.Error("expected match")
	}
	if !result.RateLimited {
		t.Error("expected rate limited")
	}
	if result.Response != nil {
		t.Error("expected no response when rate limited")
	}
}

func TestHandleRequest_LatencyPolicy(t *testing.T) {
	uc := newHandleRequestUC(true)
	req := &match.IncomingRequest{
		Method:  "GET",
		Path:    "/api/slow",
		Headers: map[string]string{},
	}
	candidates := []*match.CompiledScenario{
		{
			ID:       "slow",
			Method:   "GET",
			PathKey:  "GET:/api/slow",
			Priority: 10,
			Predicates: []match.FieldPredicate{
				{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
			},
			Response: match.CompiledResponse{Status: 200, Body: []byte("ok")},
			Policy: &match.CompiledPolicy{
				Latency: &match.CompiledLatency{FixedMs: 100, JitterMs: 50},
			},
		},
	}

	result := uc.Execute(context.Background(), req, candidates)

	if !result.Matched {
		t.Error("expected match")
	}
	if result.Response == nil {
		t.Fatal("expected response")
	}
	if result.Response.Status != 200 {
		t.Errorf("expected status 200, got %d", result.Response.Status)
	}
}

func TestHandleRequest_ContentTypeInference(t *testing.T) {
	uc := newHandleRequestUC(true)
	req := &match.IncomingRequest{
		Method:  "GET",
		Path:    "/api/infer",
		Headers: map[string]string{},
	}
	candidates := []*match.CompiledScenario{
		{
			ID:       "infer",
			Method:   "GET",
			PathKey:  "GET:/api/infer",
			Priority: 10,
			Predicates: []match.FieldPredicate{
				{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
			},
			Response: match.CompiledResponse{
				Status: 200,
				Body:   []byte(`{"hello":"world"}`),
				// ContentType intentionally empty — should be inferred.
			},
		},
	}

	result := uc.Execute(context.Background(), req, candidates)

	if !result.Matched {
		t.Fatal("expected match")
	}
	if result.Response.ContentType == "" {
		t.Error("expected content type to be inferred")
	}
}

func TestHandleRequest_LatencyCancelled(t *testing.T) {
	uc := newHandleRequestUC(true)
	req := &match.IncomingRequest{
		Method:  "GET",
		Path:    "/api/slow",
		Headers: map[string]string{},
	}
	candidates := []*match.CompiledScenario{
		{
			ID:       "slow-cancel",
			Method:   "GET",
			PathKey:  "GET:/api/slow",
			Priority: 10,
			Predicates: []match.FieldPredicate{
				{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
			},
			Response: match.CompiledResponse{Status: 200, Body: []byte("ok")},
			Policy: &match.CompiledPolicy{
				Latency: &match.CompiledLatency{FixedMs: 5000}, // 5 seconds
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	result := uc.Execute(ctx, req, candidates)

	if !result.Matched {
		t.Error("expected match even when latency cancelled")
	}
}

func TestHandleRequest_PaginationPolicy(t *testing.T) {
	uc := newHandleRequestUC(true)
	req := &match.IncomingRequest{
		Method:  "GET",
		Path:    "/api/paginated",
		Headers: map[string]string{},
	}
	candidates := []*match.CompiledScenario{
		{
			ID:       "paginated",
			Method:   "GET",
			PathKey:  "GET:/api/paginated",
			Priority: 10,
			Predicates: []match.FieldPredicate{
				{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
			},
			Response: match.CompiledResponse{Status: 200, Body: []byte("[1,2,3]")},
			Policy: &match.CompiledPolicy{
				Pagination: &match.CompiledPagination{
					Style:       "page_size",
					DefaultSize: 10,
					MaxSize:     100,
					DataPath:    "$",
				},
			},
		},
	}

	result := uc.Execute(context.Background(), req, candidates)

	if !result.Matched {
		t.Error("expected match")
	}
	if result.Pagination == nil {
		t.Error("expected pagination config in result")
	}
}

func TestHandleRequest_RateLimitDefaultKey(t *testing.T) {
	uc := newHandleRequestUC(true)
	req := &match.IncomingRequest{
		Method:  "GET",
		Path:    "/api/test",
		Headers: map[string]string{},
	}
	candidates := []*match.CompiledScenario{
		{
			ID:       "empty-key",
			Method:   "GET",
			PathKey:  "GET:/api/test",
			Priority: 10,
			Predicates: []match.FieldPredicate{
				{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
			},
			Response: match.CompiledResponse{Status: 200, Body: []byte("ok")},
			Policy: &match.CompiledPolicy{
				RateLimit: &match.CompiledRateLimit{Rate: 100, Burst: 10, Key: ""}, // empty key defaults to scenario ID
			},
		},
	}

	result := uc.Execute(context.Background(), req, candidates)

	if !result.Matched {
		t.Error("expected match")
	}
}

func TestHandleRequest_TraceEntryRecorded(t *testing.T) {
	traceBuf := trace.NewRingBuffer(50)
	uc := usecases.NewHandleRequestUseCase(
		match.NewEvaluator(),
		&testutil.FixedClock{T: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
		&testutil.StubRateLimiter{AllowAll: true},
		&testutil.NoopLogger{},
		traceBuf,
	)
	req := &match.IncomingRequest{
		Method: "GET",
		Path:   "/api/traced",
	}
	uc.Execute(context.Background(), req, nil)

	entries := traceBuf.Last(10)
	if len(entries) != 1 {
		t.Fatalf("expected 1 trace entry, got %d", len(entries))
	}
	if entries[0].Method != "GET" {
		t.Errorf("expected method GET, got %s", entries[0].Method)
	}
	if entries[0].Path != "/api/traced" {
		t.Errorf("expected path /api/traced, got %s", entries[0].Path)
	}
}
