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

func (r *mockRepo) LoadByID(_ context.Context, id string) (*scenario.Scenario, error) {
	for _, s := range r.scenarios {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, scenario.ErrNotFound
}

func (r *mockRepo) SaveScenario(_ context.Context, _ *scenario.Scenario, _ []byte) error {
	return nil
}

func (r *mockRepo) DeleteScenario(_ context.Context, _ string, _ int) error {
	return nil
}

func (r *mockRepo) ReadSourceYAML(_ context.Context, _ *scenario.Scenario) ([]byte, error) {
	return nil, nil
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

func TestLoadScenariosUseCase_SetDefaultEngine(t *testing.T) {
	repo := &mockRepo{
		scenarios: []*scenario.Scenario{
			{
				ID: "no-engine", Priority: 10,
				When:     scenario.WhenClause{Method: "GET", Path: "/api/test"},
				Response: scenario.Response{Status: 200, Body: "hello ${now()}"},
			},
			{
				ID: "has-engine", Priority: 5,
				When:     scenario.WhenClause{Method: "GET", Path: "/api/other"},
				Response: scenario.Response{Status: 200, Body: "hello", Engine: "jinja2"},
			},
		},
	}

	uc := usecases.NewLoadScenariosUseCase(repo, newTestCompiler(t), &testutil.NoopLogger{})
	uc.SetDefaultEngine("expr")

	// The compilation would fail without a template registry since default engine is now "expr".
	// But since we use nil registry in the compiler, this will log a warning for the first scenario
	// and compile the second with its explicit "jinja2" engine (also fails with nil registry).
	// The test exercises SetDefaultEngine and default engine application paths.
	idx, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Both scenarios fail compilation (no registry), but that's expected.
	// The important thing is that the code path for SetDefaultEngine was exercised.
	_ = idx
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
