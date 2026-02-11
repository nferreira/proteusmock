package services_test

import (
	"testing"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
)

func TestScenarioIndex_Lookup(t *testing.T) {
	idx := services.NewScenarioIndex()

	idx.Add(&match.CompiledScenario{
		ID:       "a",
		Method:   "GET",
		PathKey:  "GET:/api/items",
		Priority: 10,
	})
	idx.Add(&match.CompiledScenario{
		ID:       "b",
		Method:   "GET",
		PathKey:  "GET:/api/items",
		Priority: 20,
	})
	idx.Add(&match.CompiledScenario{
		ID:       "c",
		Method:   "POST",
		PathKey:  "POST:/api/items",
		Priority: 5,
	})

	idx.Build()

	candidates := idx.Lookup("GET:/api/items")
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	// Higher priority first.
	if candidates[0].ID != "b" {
		t.Errorf("expected 'b' first, got %q", candidates[0].ID)
	}
	if candidates[1].ID != "a" {
		t.Errorf("expected 'a' second, got %q", candidates[1].ID)
	}

	postCandidates := idx.Lookup("POST:/api/items")
	if len(postCandidates) != 1 {
		t.Fatalf("expected 1 POST candidate, got %d", len(postCandidates))
	}
}

func TestScenarioIndex_DeterministicOrdering(t *testing.T) {
	idx := services.NewScenarioIndex()

	idx.Add(&match.CompiledScenario{ID: "z", Method: "GET", PathKey: "GET:/test", Priority: 10})
	idx.Add(&match.CompiledScenario{ID: "a", Method: "GET", PathKey: "GET:/test", Priority: 10})
	idx.Add(&match.CompiledScenario{ID: "m", Method: "GET", PathKey: "GET:/test", Priority: 10})

	idx.Build()

	candidates := idx.Lookup("GET:/test")
	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(candidates))
	}
	if candidates[0].ID != "a" {
		t.Errorf("expected 'a' first, got %q", candidates[0].ID)
	}
	if candidates[1].ID != "m" {
		t.Errorf("expected 'm' second, got %q", candidates[1].ID)
	}
	if candidates[2].ID != "z" {
		t.Errorf("expected 'z' third, got %q", candidates[2].ID)
	}
}

func TestScenarioIndex_SpecificityTiebreaker(t *testing.T) {
	idx := services.NewScenarioIndex()

	// Less specific: 1 predicate.
	idx.Add(&match.CompiledScenario{
		ID:       "generic",
		Method:   "POST",
		PathKey:  "POST:/api/items",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "header:Content-Type"},
		},
	})
	// More specific: 2 predicates.
	idx.Add(&match.CompiledScenario{
		ID:       "specific",
		Method:   "POST",
		PathKey:  "POST:/api/items",
		Priority: 10,
		Predicates: []match.FieldPredicate{
			{Field: "header:Content-Type"},
			{Field: "header:X-Api-Key"},
		},
	})

	idx.Build()

	candidates := idx.Lookup("POST:/api/items")
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].ID != "specific" {
		t.Errorf("expected 'specific' first (more predicates), got %q", candidates[0].ID)
	}
	if candidates[1].ID != "generic" {
		t.Errorf("expected 'generic' second (fewer predicates), got %q", candidates[1].ID)
	}
}

func TestScenarioIndex_Paths(t *testing.T) {
	idx := services.NewScenarioIndex()

	idx.Add(&match.CompiledScenario{ID: "a", Method: "GET", PathKey: "GET:/api/items"})
	idx.Add(&match.CompiledScenario{ID: "b", Method: "POST", PathKey: "POST:/api/items"})
	idx.Add(&match.CompiledScenario{ID: "c", Method: "GET", PathKey: "GET:/api/health"})

	idx.Build()

	paths := idx.Paths()
	if len(paths) != 2 {
		t.Fatalf("expected 2 unique paths, got %d: %v", len(paths), paths)
	}
}

func TestScenarioIndex_Empty(t *testing.T) {
	idx := services.NewScenarioIndex()
	idx.Build()

	if len(idx.Lookup("GET:/nothing")) != 0 {
		t.Error("expected empty lookup")
	}
	if len(idx.All()) != 0 {
		t.Error("expected empty All()")
	}
	if len(idx.Keys()) != 0 {
		t.Error("expected empty Keys()")
	}
}

func TestScenarioIndex_AllAndKeys(t *testing.T) {
	idx := services.NewScenarioIndex()

	idx.Add(&match.CompiledScenario{ID: "a", Method: "GET", PathKey: "GET:/api/items"})
	idx.Add(&match.CompiledScenario{ID: "b", Method: "POST", PathKey: "POST:/api/items"})
	idx.Add(&match.CompiledScenario{ID: "c", Method: "GET", PathKey: "GET:/api/health"})

	idx.Build()

	all := idx.All()
	if len(all) != 3 {
		t.Errorf("expected 3 scenarios from All(), got %d", len(all))
	}

	keys := idx.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
	// Keys should be sorted.
	for i := 1; i < len(keys); i++ {
		if keys[i] < keys[i-1] {
			t.Errorf("keys not sorted: %v", keys)
			break
		}
	}
}
