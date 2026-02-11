package usecases_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/sophialabs/proteusmock/internal/domain/scenario"
	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
	"github.com/sophialabs/proteusmock/internal/infrastructure/usecases"
	"github.com/sophialabs/proteusmock/internal/testutil"
)

type mockRepo struct {
	scenarios []*scenario.Scenario
	err       error
}

func (r *mockRepo) LoadAll(_ context.Context) ([]*scenario.Scenario, error) {
	return r.scenarios, r.err
}

func newTestCompiler(t *testing.T) *services.Compiler {
	t.Helper()
	c, err := services.NewCompiler(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("NewCompiler failed: %v", err)
	}
	return c
}

func TestLoadScenariosUseCase_Success(t *testing.T) {
	repo := &mockRepo{
		scenarios: []*scenario.Scenario{
			{
				ID: "s1", Name: "S1", Priority: 10,
				When:     scenario.WhenClause{Method: "GET", Path: "/api/health"},
				Response: scenario.Response{Status: 200, Body: "ok"},
			},
			{
				ID: "s2", Name: "S2", Priority: 5,
				When:     scenario.WhenClause{Method: "POST", Path: "/api/items"},
				Response: scenario.Response{Status: 201, Body: "created"},
			},
		},
	}

	uc := usecases.NewLoadScenariosUseCase(repo, newTestCompiler(t), &testutil.NoopLogger{})
	idx, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(idx.All()) != 2 {
		t.Errorf("expected 2 compiled scenarios, got %d", len(idx.All()))
	}

	candidates := idx.Lookup("GET:/api/health")
	if len(candidates) != 1 {
		t.Errorf("expected 1 candidate for GET:/api/health, got %d", len(candidates))
	}
}

func TestLoadScenariosUseCase_DuplicateID(t *testing.T) {
	repo := &mockRepo{
		scenarios: []*scenario.Scenario{
			{ID: "dup", When: scenario.WhenClause{Method: "GET", Path: "/a"}, Response: scenario.Response{Status: 200}},
			{ID: "dup", When: scenario.WhenClause{Method: "GET", Path: "/b"}, Response: scenario.Response{Status: 200}},
		},
	}

	uc := usecases.NewLoadScenariosUseCase(repo, newTestCompiler(t), &testutil.NoopLogger{})
	_, err := uc.Execute(context.Background())
	if err == nil {
		t.Error("expected error for duplicate IDs")
	}
}

func TestLoadScenariosUseCase_RepoError(t *testing.T) {
	repo := &mockRepo{err: fmt.Errorf("disk error")}

	uc := usecases.NewLoadScenariosUseCase(repo, newTestCompiler(t), &testutil.NoopLogger{})
	_, err := uc.Execute(context.Background())
	if err == nil {
		t.Error("expected error from repo failure")
	}
}

func TestLoadScenariosUseCase_PartialCompileFailure(t *testing.T) {
	repo := &mockRepo{
		scenarios: []*scenario.Scenario{
			{
				ID: "good", Priority: 10,
				When:     scenario.WhenClause{Method: "GET", Path: "/ok"},
				Response: scenario.Response{Status: 200, Body: "ok"},
			},
			{
				ID: "bad-regex", Priority: 5,
				When: scenario.WhenClause{
					Method: "GET", Path: "/bad",
					Headers: map[string]scenario.StringMatcher{
						"X-Bad": {Pattern: "[invalid"},
					},
				},
				Response: scenario.Response{Status: 200},
			},
		},
	}

	uc := usecases.NewLoadScenariosUseCase(repo, newTestCompiler(t), &testutil.NoopLogger{})
	idx, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Only the good scenario should be in the index.
	if len(idx.All()) != 1 {
		t.Errorf("expected 1 compiled scenario (partial failure), got %d", len(idx.All()))
	}
}
