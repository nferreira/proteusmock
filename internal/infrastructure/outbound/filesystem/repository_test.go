package filesystem_test

import (
	"context"
	"testing"

	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/filesystem"
)

func newTestRepo(t *testing.T, rootDir string) *filesystem.YAMLRepository {
	t.Helper()
	repo, err := filesystem.NewYAMLRepository(rootDir)
	if err != nil {
		t.Fatalf("NewYAMLRepository failed: %v", err)
	}
	return repo
}

func TestYAMLRepository_LoadAll_SimpleScenario(t *testing.T) {
	repo := newTestRepo(t, "../../../../testdata")
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) == 0 {
		t.Fatal("expected at least one scenario")
	}

	// Find the simple-get scenario.
	var found bool
	for _, s := range scenarios {
		if s.ID == "simple-get" {
			found = true
			if s.Name != "Simple GET endpoint" {
				t.Errorf("unexpected name: %s", s.Name)
			}
			if s.Priority != 10 {
				t.Errorf("unexpected priority: %d", s.Priority)
			}
			if s.When.Method != "GET" {
				t.Errorf("unexpected method: %s", s.When.Method)
			}
			if s.When.Path != "/api/v1/health" {
				t.Errorf("unexpected path: %s", s.When.Path)
			}
			if s.Response.Status != 200 {
				t.Errorf("unexpected status: %d", s.Response.Status)
			}
		}
	}
	if !found {
		t.Error("simple-get scenario not found")
	}
}

func TestYAMLRepository_LoadAll_WithBody(t *testing.T) {
	repo := newTestRepo(t, "../../../../testdata")
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	for _, s := range scenarios {
		if s.ID == "post-properties" {
			if s.When.Body == nil {
				t.Fatal("expected body clause")
			}
			if s.When.Body.ContentType != "json" {
				t.Errorf("unexpected content type: %s", s.When.Body.ContentType)
			}
			if len(s.When.Body.Conditions) != 1 {
				t.Fatalf("expected 1 body condition, got %d", len(s.When.Body.Conditions))
			}
			cond := s.When.Body.Conditions[0]
			if cond.Extractor != "$.method.params.contract_id" {
				t.Errorf("unexpected extractor: %s", cond.Extractor)
			}
			if !cond.Matcher.IsExact() || cond.Matcher.Value() != "100100" {
				t.Errorf("unexpected matcher: exact=%v, value=%s", cond.Matcher.IsExact(), cond.Matcher.Value())
			}
			// Check header exact match.
			ct, ok := s.When.Headers["Content-Type"]
			if !ok {
				t.Fatal("expected Content-Type header matcher")
			}
			if !ct.IsExact() || ct.Value() != "application/json" {
				t.Errorf("unexpected header matcher: exact=%v, value=%s", ct.IsExact(), ct.Value())
			}
			return
		}
	}
	t.Error("post-properties scenario not found")
}

func TestYAMLRepository_LoadAll_WithPolicy(t *testing.T) {
	repo := newTestRepo(t, "../../../../testdata")
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	for _, s := range scenarios {
		if s.ID == "rate-limited" {
			if s.Policy == nil {
				t.Fatal("expected policy")
			}
			if s.Policy.RateLimit == nil {
				t.Fatal("expected rate limit")
			}
			if s.Policy.RateLimit.Rate != 10.0 {
				t.Errorf("unexpected rate: %f", s.Policy.RateLimit.Rate)
			}
			if s.Policy.RateLimit.Burst != 5 {
				t.Errorf("unexpected burst: %d", s.Policy.RateLimit.Burst)
			}
			if s.Policy.Latency == nil {
				t.Fatal("expected latency")
			}
			if s.Policy.Latency.FixedMs != 100 {
				t.Errorf("unexpected fixed_ms: %d", s.Policy.Latency.FixedMs)
			}
			return
		}
	}
	t.Error("rate-limited scenario not found")
}

func TestYAMLRepository_LoadAll_MultiScenarioFile(t *testing.T) {
	repo := newTestRepo(t, "../../../../testdata")
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	var foundOne, foundTwo bool
	for _, s := range scenarios {
		switch s.ID {
		case "multi-one":
			foundOne = true
		case "multi-two":
			foundTwo = true
			if len(s.When.Headers) == 0 {
				t.Error("expected headers for multi-two")
			}
		}
	}

	if !foundOne {
		t.Error("multi-one not found")
	}
	if !foundTwo {
		t.Error("multi-two not found")
	}
}

func TestYAMLRepository_LoadAll_IncludeResolution(t *testing.T) {
	repo := newTestRepo(t, "../../../../testdata")
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	for _, s := range scenarios {
		if s.ID == "include-test" {
			if s.Response.Body == "" {
				t.Error("expected body to be resolved from include")
			}
			if s.Response.Body != `{"status": "healthy", "version": "1.0.0"}` {
				t.Errorf("unexpected body: %s", s.Response.Body)
			}
			return
		}
	}
	t.Error("include-test scenario not found")
}

func TestYAMLRepository_LoadAll_NonexistentDir(t *testing.T) {
	repo := newTestRepo(t, "/nonexistent/path")
	_, err := repo.LoadAll(context.Background())
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}
