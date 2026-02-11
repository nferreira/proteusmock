package match_test

import (
	"testing"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

func TestEvaluator_NoCandidate(t *testing.T) {
	eval := match.NewEvaluator()
	req := &match.IncomingRequest{
		Method: "GET",
		Path:   "/test",
	}

	result := eval.Evaluate(req, nil)
	if result.Matched != nil {
		t.Error("expected no match")
	}
	if len(result.Candidates) != 0 {
		t.Errorf("expected 0 candidates, got %d", len(result.Candidates))
	}
}

func TestEvaluator_SingleMatch(t *testing.T) {
	eval := match.NewEvaluator()
	req := &match.IncomingRequest{
		Method: "GET",
		Path:   "/api/health",
	}

	candidates := []*match.CompiledScenario{
		{
			ID:       "health",
			Name:     "Health Check",
			Priority: 10,
			Predicates: []match.FieldPredicate{
				{Field: "method", Predicate: func(s string) bool { return s == "GET" }},
			},
			Response: match.CompiledResponse{Status: 200},
		},
	}

	result := eval.Evaluate(req, candidates)
	if result.Matched == nil {
		t.Fatal("expected a match")
	}
	if result.Matched.ID != "health" {
		t.Errorf("expected match ID 'health', got %q", result.Matched.ID)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(result.Candidates))
	}
	if !result.Candidates[0].Matched {
		t.Error("expected candidate to be matched")
	}
}

func TestEvaluator_PriorityOrdering(t *testing.T) {
	eval := match.NewEvaluator()
	req := &match.IncomingRequest{
		Method: "GET",
		Path:   "/api/items",
	}

	always := func(string) bool { return true }

	// Pre-sorted: higher priority first (as ScenarioIndex.Build produces).
	candidates := []*match.CompiledScenario{
		{
			ID:         "high-priority",
			Name:       "High Priority",
			Priority:   20,
			Predicates: []match.FieldPredicate{{Field: "method", Predicate: always}},
			Response:   match.CompiledResponse{Status: 200, Body: []byte("high")},
		},
		{
			ID:         "low-priority",
			Name:       "Low Priority",
			Priority:   5,
			Predicates: []match.FieldPredicate{{Field: "method", Predicate: always}},
			Response:   match.CompiledResponse{Status: 200, Body: []byte("low")},
		},
	}

	result := eval.Evaluate(req, candidates)
	if result.Matched == nil {
		t.Fatal("expected a match")
	}
	if result.Matched.ID != "high-priority" {
		t.Errorf("expected 'high-priority' to win, got %q", result.Matched.ID)
	}
}

func TestEvaluator_FailedPredicateTrace(t *testing.T) {
	eval := match.NewEvaluator()
	req := &match.IncomingRequest{
		Method:  "POST",
		Path:    "/api/items",
		Headers: map[string]string{"Content-Type": "text/plain"},
	}

	candidates := []*match.CompiledScenario{
		{
			ID:       "needs-json",
			Name:     "Needs JSON",
			Priority: 10,
			Predicates: []match.FieldPredicate{
				{Field: "method", Predicate: func(s string) bool { return s == "POST" }},
				{Field: "header:Content-Type", Predicate: func(s string) bool { return s == "application/json" }},
			},
			Response: match.CompiledResponse{Status: 200},
		},
	}

	result := eval.Evaluate(req, candidates)
	if result.Matched != nil {
		t.Error("expected no match")
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(result.Candidates))
	}
	c := result.Candidates[0]
	if c.Matched {
		t.Error("expected candidate to not match")
	}
	if c.FailedField != "header:Content-Type" {
		t.Errorf("expected failed field 'header:Content-Type', got %q", c.FailedField)
	}
}

func TestEvaluator_BodyFieldPredicate(t *testing.T) {
	eval := match.NewEvaluator()
	req := &match.IncomingRequest{
		Method: "POST",
		Path:   "/api/items",
		Body:   []byte(`{"name":"test"}`),
	}

	candidates := []*match.CompiledScenario{
		{
			ID:       "body-match",
			Name:     "Body Match",
			Priority: 10,
			Predicates: []match.FieldPredicate{
				{Field: "body", Predicate: func(s string) bool { return s == `{"name":"test"}` }},
			},
			Response: match.CompiledResponse{Status: 200},
		},
	}

	result := eval.Evaluate(req, candidates)
	if result.Matched == nil {
		t.Fatal("expected a match for field 'body'")
	}
	if result.Matched.ID != "body-match" {
		t.Errorf("expected match ID 'body-match', got %q", result.Matched.ID)
	}
}

func TestEvaluator_DeterministicIDOrdering(t *testing.T) {
	eval := match.NewEvaluator()
	req := &match.IncomingRequest{Method: "GET", Path: "/"}

	always := func(string) bool { return true }

	// Pre-sorted: same priority, ID ascending (as ScenarioIndex.Build produces).
	candidates := []*match.CompiledScenario{
		{
			ID:         "a-scenario",
			Priority:   10,
			Predicates: []match.FieldPredicate{{Field: "method", Predicate: always}},
			Response:   match.CompiledResponse{Status: 200},
		},
		{
			ID:         "b-scenario",
			Priority:   10,
			Predicates: []match.FieldPredicate{{Field: "method", Predicate: always}},
			Response:   match.CompiledResponse{Status: 200},
		},
	}

	result := eval.Evaluate(req, candidates)
	if result.Matched == nil {
		t.Fatal("expected a match")
	}
	if result.Matched.ID != "a-scenario" {
		t.Errorf("expected 'a-scenario' (first in pre-sorted order), got %q", result.Matched.ID)
	}
}
