package filesystem_test

import (
	"context"
	"os"
	"path/filepath"
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

func TestYAMLRepository_LoadAll_NonexistentDir(t *testing.T) {
	repo := newTestRepo(t, "/nonexistent/path")
	_, err := repo.LoadAll(context.Background())
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestYAMLRepository_LoadAll_InvalidYAML(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(":\n  :\n\t\t\tinvalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := newTestRepo(t, dir)
	_, err := repo.LoadAll(context.Background())
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestYAMLRepository_LoadAll_PaginationWithCustomEnvelope(t *testing.T) {
	dir := t.TempDir()

	content := `
id: custom-pagination
name: Custom pagination
priority: 10
when:
  method: GET
  path: /api/custom
policy:
  pagination:
    style: offset_limit
    offset_param: start
    limit_param: count
    default_size: 20
    max_size: 50
    data_path: "$.results"
    envelope:
      data_field: items
      total_items_field: total
      total_pages_field: pages
      page_field: current_page
      size_field: per_page
      has_next_field: more
      has_previous_field: less
response:
  status: 200
  body: '{"results": []}'
`
	if err := os.WriteFile(filepath.Join(dir, "pagination.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := newTestRepo(t, dir)
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}

	s := scenarios[0]
	p := s.Policy.Pagination
	if p.Style != "offset_limit" {
		t.Errorf("expected offset_limit style, got %q", string(p.Style))
	}
	if p.Envelope.DataField != "items" {
		t.Errorf("expected data_field 'items', got %q", p.Envelope.DataField)
	}
	if p.Envelope.TotalItemsField != "total" {
		t.Errorf("expected total_items_field 'total', got %q", p.Envelope.TotalItemsField)
	}
}

func TestYAMLRepository_LoadAll_PaginationInvalidStyle(t *testing.T) {
	dir := t.TempDir()

	content := `
id: bad-style
name: Bad pagination style
priority: 10
when:
  method: GET
  path: /api/test
policy:
  pagination:
    style: unknown_style
response:
  status: 200
  body: '[]'
`
	if err := os.WriteFile(filepath.Join(dir, "bad_style.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := newTestRepo(t, dir)
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}

	// Invalid style should default to page_size.
	if scenarios[0].Policy.Pagination.Style != "page_size" {
		t.Errorf("expected default style 'page_size', got %q", string(scenarios[0].Policy.Pagination.Style))
	}
}

func TestYAMLRepository_LoadAll_BodyCombinators(t *testing.T) {
	dir := t.TempDir()

	content := `
id: combinators
name: Body combinators
priority: 10
when:
  method: POST
  path: /api/test
  body:
    content_type: json
    all:
      - content_type: json
        conditions:
          - extractor: "$.name"
            matcher: "=Alice"
    any:
      - content_type: json
        conditions:
          - extractor: "$.type"
            matcher: "=A"
      - content_type: json
        conditions:
          - extractor: "$.type"
            matcher: "=B"
    not:
      content_type: json
      conditions:
        - extractor: "$.status"
          matcher: "=blocked"
response:
  status: 200
`
	if err := os.WriteFile(filepath.Join(dir, "combinators.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := newTestRepo(t, dir)
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}

	s := scenarios[0]
	if s.When.Body == nil {
		t.Fatal("expected body clause")
	}
	if len(s.When.Body.All) != 1 {
		t.Errorf("expected 1 all clause, got %d", len(s.When.Body.All))
	}
	if len(s.When.Body.Any) != 2 {
		t.Errorf("expected 2 any clauses, got %d", len(s.When.Body.Any))
	}
	if s.When.Body.Not == nil {
		t.Error("expected not clause")
	}
}

func TestYAMLRepository_LoadAll_EngineField(t *testing.T) {
	dir := t.TempDir()

	content := `
id: template-test
name: Template test
priority: 10
when:
  method: GET
  path: /api/test
response:
  status: 200
  body: 'Hello ${name}'
  engine: expr
`
	if err := os.WriteFile(filepath.Join(dir, "template.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := newTestRepo(t, dir)
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}

	if scenarios[0].Response.Engine != "expr" {
		t.Errorf("expected engine 'expr', got %q", scenarios[0].Response.Engine)
	}
}

func TestYAMLRepository_LoadAll_IgnoresNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a non-YAML file that should be ignored.
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{}`), 0o644)

	content := `
id: only-yaml
name: Only YAML
priority: 10
when:
  method: GET
  path: /test
response:
  status: 200
`
	os.WriteFile(filepath.Join(dir, "scenario.yaml"), []byte(content), 0o644)

	repo := newTestRepo(t, dir)
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Errorf("expected 1 scenario (only YAML), got %d", len(scenarios))
	}
}

func TestYAMLRepository_LoadAll_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	repo := newTestRepo(t, dir)
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) != 0 {
		t.Errorf("expected 0 scenarios, got %d", len(scenarios))
	}
}

func TestYAMLRepository_LoadAll_DecodeError(t *testing.T) {
	dir := t.TempDir()

	// Valid YAML but can't be decoded as a scenario (numeric value for when).
	content := `
id: decode-fail
when: 42
response: not-a-map
`
	os.WriteFile(filepath.Join(dir, "decode_fail.yaml"), []byte(content), 0o644)

	repo := newTestRepo(t, dir)
	_, err := repo.LoadAll(context.Background())
	if err == nil {
		t.Error("expected error for invalid scenario structure")
	}
}

func TestYAMLRepository_LoadAll_NilPolicy(t *testing.T) {
	dir := t.TempDir()

	content := `
id: no-policy
name: No policy
priority: 10
when:
  method: GET
  path: /test
response:
  status: 200
`
	os.WriteFile(filepath.Join(dir, "no_policy.yaml"), []byte(content), 0o644)

	repo := newTestRepo(t, dir)
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}
	if scenarios[0].Policy != nil {
		t.Error("expected nil policy")
	}
}

func TestYAMLRepository_LoadAll_PaginationNilEnvelope(t *testing.T) {
	dir := t.TempDir()

	content := `
id: nil-envelope
name: Nil envelope
priority: 10
when:
  method: GET
  path: /test
policy:
  pagination:
    style: page_size
response:
  status: 200
  body: '[]'
`
	os.WriteFile(filepath.Join(dir, "nil_envelope.yaml"), []byte(content), 0o644)

	repo := newTestRepo(t, dir)
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}

	p := scenarios[0].Policy.Pagination
	// Should have default envelope values.
	if p.Envelope.DataField != "data" {
		t.Errorf("expected default data field 'data', got %q", p.Envelope.DataField)
	}
}

func TestYAMLRepository_LoadAll_BodyFile(t *testing.T) {
	dir := t.TempDir()

	content := `
id: bodyfile
name: Body file test
priority: 10
when:
  method: GET
  path: /test
response:
  status: 200
  body_file: response.json
  content_type: application/json
`
	os.WriteFile(filepath.Join(dir, "scenario.yaml"), []byte(content), 0o644)

	repo := newTestRepo(t, dir)
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}
	if scenarios[0].Response.BodyFile != "response.json" {
		t.Errorf("expected body_file 'response.json', got %q", scenarios[0].Response.BodyFile)
	}
}

func TestYAMLRepository_LoadAll_PolicyWithLatencyOnly(t *testing.T) {
	dir := t.TempDir()

	content := `
id: latency-only
name: Latency only
priority: 10
when:
  method: GET
  path: /test
policy:
  latency:
    fixed_ms: 100
    jitter_ms: 20
response:
  status: 200
`
	os.WriteFile(filepath.Join(dir, "latency.yaml"), []byte(content), 0o644)

	repo := newTestRepo(t, dir)
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}

	p := scenarios[0].Policy
	if p.RateLimit != nil {
		t.Error("expected nil rate limit")
	}
	if p.Latency == nil {
		t.Fatal("expected latency")
	}
	if p.Latency.FixedMs != 100 {
		t.Errorf("expected fixed_ms 100, got %d", p.Latency.FixedMs)
	}
}

func TestYAMLRepository_LoadAll_InvalidScenarioInList(t *testing.T) {
	dir := t.TempDir()

	// A list with one invalid entry (when is a number, not a map).
	content := `
- id: valid
  when:
    method: GET
    path: /test
  response:
    status: 200
- id: bad
  when: 42
  response: "not a map"
`
	os.WriteFile(filepath.Join(dir, "list_bad.yaml"), []byte(content), 0o644)

	repo := newTestRepo(t, dir)
	_, err := repo.LoadAll(context.Background())
	if err == nil {
		t.Error("expected error for invalid scenario in list")
	}
}

func TestYAMLRepository_LoadAll_HeaderPatternMatcher(t *testing.T) {
	dir := t.TempDir()

	content := `
id: pattern-test
name: Pattern header
priority: 10
when:
  method: GET
  path: /test
  headers:
    X-Api-Key: "secret-.*"
response:
  status: 200
`
	os.WriteFile(filepath.Join(dir, "pattern.yaml"), []byte(content), 0o644)

	repo := newTestRepo(t, dir)
	scenarios, err := repo.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}

	hdr := scenarios[0].When.Headers["X-Api-Key"]
	if hdr.IsExact() {
		t.Error("expected pattern matcher, got exact")
	}
	if hdr.Pattern != "secret-.*" {
		t.Errorf("expected pattern 'secret-.*', got %q", hdr.Pattern)
	}
}
