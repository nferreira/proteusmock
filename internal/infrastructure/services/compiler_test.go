package services_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/domain/scenario"
	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
)

func newTestCompiler(t *testing.T) *services.Compiler {
	t.Helper()
	c, err := services.NewCompiler(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("NewCompiler failed: %v", err)
	}
	return c
}

func TestCompiler_SimpleScenario(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID:       "test-1",
		Name:     "Test",
		Priority: 10,
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/api/health",
		},
		Response: scenario.Response{
			Status: 200,
			Body:   `{"ok": true}`,
		},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	if cs.ID != "test-1" {
		t.Errorf("unexpected ID: %s", cs.ID)
	}
	if cs.PathKey != "GET:/api/health" {
		t.Errorf("unexpected PathKey: %s", cs.PathKey)
	}
	if cs.Response.Status != 200 {
		t.Errorf("unexpected status: %d", cs.Response.Status)
	}
	if string(cs.Response.Body) != `{"ok": true}` {
		t.Errorf("unexpected body: %s", cs.Response.Body)
	}

	// Method predicate should match GET.
	for _, p := range cs.Predicates {
		if p.Field == "method" {
			if !p.Predicate("GET") {
				t.Error("method predicate should match GET")
			}
			if p.Predicate("POST") {
				t.Error("method predicate should not match POST")
			}
		}
	}
}

func TestCompiler_ExactHeaderMatcher(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "hdr-test",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/api/test",
			Headers: map[string]scenario.StringMatcher{
				"Content-Type": {Exact: "application/json"},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	for _, p := range cs.Predicates {
		if p.Field == "header:Content-Type" {
			if !p.Predicate("application/json") {
				t.Error("should match application/json")
			}
			if p.Predicate("text/plain") {
				t.Error("should not match text/plain")
			}
			return
		}
	}
	t.Error("header predicate not found")
}

func TestCompiler_RegexHeaderMatcher(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "regex-hdr",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/api/test",
			Headers: map[string]scenario.StringMatcher{
				"X-Api-Key": {Pattern: "secret-.*"},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	for _, p := range cs.Predicates {
		if p.Field == "header:X-Api-Key" {
			if !p.Predicate("secret-abc123") {
				t.Error("should match secret-abc123")
			}
			if p.Predicate("public-key") {
				t.Error("should not match public-key")
			}
			return
		}
	}
	t.Error("header predicate not found")
}

func TestCompiler_JSONPathBody(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "json-body",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/api/query",
			Body: &scenario.BodyClause{
				ContentType: "json",
				Conditions: []scenario.BodyCondition{
					{
						Extractor: "$.name",
						Matcher:   scenario.StringMatcher{Exact: "Alice"},
					},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	for _, p := range cs.Predicates {
		if p.Field == "body:$.name" {
			if !p.Predicate(`{"name": "Alice"}`) {
				t.Error("should match JSON with name=Alice")
			}
			if p.Predicate(`{"name": "Bob"}`) {
				t.Error("should not match JSON with name=Bob")
			}
			return
		}
	}
	t.Error("body predicate not found")
}

func TestCompiler_XPathBody(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "xml-body",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/api/xml",
			Body: &scenario.BodyClause{
				ContentType: "xml",
				Conditions: []scenario.BodyCondition{
					{
						Extractor: "//user/name",
						Matcher:   scenario.StringMatcher{Exact: "Alice"},
					},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	for _, p := range cs.Predicates {
		if p.Field == "body://user/name" {
			xml := `<user><name>Alice</name></user>`
			if !p.Predicate(xml) {
				t.Error("should match XML with name=Alice")
			}
			xml = `<user><name>Bob</name></user>`
			if p.Predicate(xml) {
				t.Error("should not match XML with name=Bob")
			}
			return
		}
	}
	t.Error("body predicate not found")
}

func TestCompiler_InvalidRegex(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "bad-regex",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
			Headers: map[string]scenario.StringMatcher{
				"X-Bad": {Pattern: "[invalid"},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	_, err := compiler.CompileScenario(s)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestCompiler_BooleanCombinators(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "bool-test",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/api/complex",
			Body: &scenario.BodyClause{
				ContentType: "json",
				Any: []scenario.BodyClause{
					{
						ContentType: "json",
						Conditions: []scenario.BodyCondition{
							{Extractor: "$.type", Matcher: scenario.StringMatcher{Exact: "A"}},
						},
					},
					{
						ContentType: "json",
						Conditions: []scenario.BodyCondition{
							{Extractor: "$.type", Matcher: scenario.StringMatcher{Exact: "B"}},
						},
					},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	var anyPred func(string) bool
	for _, p := range cs.Predicates {
		if p.Field == "body:any" {
			anyPred = p.Predicate
			break
		}
	}

	if anyPred == nil {
		t.Fatal("body:any predicate not found")
	}

	if !anyPred(`{"type": "A"}`) {
		t.Error("should match type=A")
	}
	if !anyPred(`{"type": "B"}`) {
		t.Error("should match type=B")
	}
	if anyPred(`{"type": "C"}`) {
		t.Error("should not match type=C")
	}
}

func TestCompiler_DefaultStatus(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "no-status",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
		},
		Response: scenario.Response{Body: "ok"},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	if cs.Response.Status != 200 {
		t.Errorf("expected default status 200, got %d", cs.Response.Status)
	}
}

func TestCompiler_Policy(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "with-policy",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
		},
		Response: scenario.Response{Status: 200},
		Policy: &scenario.Policy{
			RateLimit: &scenario.RateLimit{Rate: 5, Burst: 10, Key: "ip"},
			Latency:   &scenario.Latency{FixedMs: 200, JitterMs: 50},
		},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	if cs.Policy == nil {
		t.Fatal("expected policy")
	}
	if cs.Policy.RateLimit.Rate != 5 {
		t.Errorf("unexpected rate: %f", cs.Policy.RateLimit.Rate)
	}
	if cs.Policy.Latency.FixedMs != 200 {
		t.Errorf("unexpected fixed_ms: %d", cs.Policy.Latency.FixedMs)
	}
}

func TestCompiler_NotCombinator(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "not-test",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/api/test",
			Body: &scenario.BodyClause{
				ContentType: "json",
				Not: &scenario.BodyClause{
					ContentType: "json",
					Conditions: []scenario.BodyCondition{
						{Extractor: "$.type", Matcher: scenario.StringMatcher{Exact: "admin"}},
					},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	var notPred func(string) bool
	for _, p := range cs.Predicates {
		if p.Field == "body:not" {
			notPred = p.Predicate
			break
		}
	}
	if notPred == nil {
		t.Fatal("body:not predicate not found")
	}

	if notPred(`{"type":"admin"}`) {
		t.Error("should NOT match type=admin")
	}
	if !notPred(`{"type":"user"}`) {
		t.Error("should match type=user")
	}
}

func TestCompiler_AllCombinator(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "all-test",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/api/test",
			Body: &scenario.BodyClause{
				ContentType: "json",
				All: []scenario.BodyClause{
					{
						ContentType: "json",
						Conditions: []scenario.BodyCondition{
							{Extractor: "$.name", Matcher: scenario.StringMatcher{Exact: "Alice"}},
						},
					},
					{
						ContentType: "json",
						Conditions: []scenario.BodyCondition{
							{Extractor: "$.age", Matcher: scenario.StringMatcher{Exact: "30"}},
						},
					},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	var allPred func(string) bool
	for _, p := range cs.Predicates {
		if p.Field == "body:all" {
			allPred = p.Predicate
			break
		}
	}
	if allPred == nil {
		t.Fatal("body:all predicate not found")
	}

	if !allPred(`{"name":"Alice","age":"30"}`) {
		t.Error("should match both conditions")
	}
	if allPred(`{"name":"Alice","age":"25"}`) {
		t.Error("should not match when only one condition passes")
	}
}

func TestCompiler_DefaultContentTypeRawBody(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "raw-body",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/api/test",
			Body: &scenario.BodyClause{
				// No content_type — defaults to raw body match.
				Conditions: []scenario.BodyCondition{
					{Extractor: "ignored", Matcher: scenario.StringMatcher{Pattern: "hello.*"}},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	var bodyPred func(string) bool
	for _, p := range cs.Predicates {
		if p.Field == "body" {
			bodyPred = p.Predicate
			break
		}
	}
	if bodyPred == nil {
		t.Fatal("body predicate not found")
	}

	if !bodyPred("hello world") {
		t.Error("should match raw body")
	}
	if bodyPred("goodbye") {
		t.Error("should not match non-matching body")
	}
}

func TestCompiler_EmptyPatternAlwaysMatches(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "empty-pattern",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
			Headers: map[string]scenario.StringMatcher{
				"X-Optional": {}, // empty exact and empty pattern
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	for _, p := range cs.Predicates {
		if p.Field == "header:X-Optional" {
			if !p.Predicate("anything") {
				t.Error("empty matcher should always match")
			}
			if !p.Predicate("") {
				t.Error("empty matcher should match empty string")
			}
			return
		}
	}
	t.Error("header predicate not found")
}

func TestCompiler_JSONPathInvalidJSON(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "jsonpath-invalid",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/test",
			Body: &scenario.BodyClause{
				ContentType: "json",
				Conditions: []scenario.BodyCondition{
					{Extractor: "$.name", Matcher: scenario.StringMatcher{Exact: "test"}},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	for _, p := range cs.Predicates {
		if p.Field == "body:$.name" {
			if p.Predicate("not json") {
				t.Error("should not match invalid JSON")
			}
			return
		}
	}
	t.Error("body predicate not found")
}

func TestCompiler_JSONPathMissingField(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "jsonpath-missing",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/test",
			Body: &scenario.BodyClause{
				ContentType: "json",
				Conditions: []scenario.BodyCondition{
					{Extractor: "$.nonexistent", Matcher: scenario.StringMatcher{Exact: "val"}},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	for _, p := range cs.Predicates {
		if p.Field == "body:$.nonexistent" {
			if p.Predicate(`{"name":"test"}`) {
				t.Error("should not match when field is missing")
			}
			return
		}
	}
	t.Error("body predicate not found")
}

func TestCompiler_XPathInvalidXML(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "xpath-invalid",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/test",
			Body: &scenario.BodyClause{
				ContentType: "xml",
				Conditions: []scenario.BodyCondition{
					{Extractor: "//name", Matcher: scenario.StringMatcher{Exact: "test"}},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	for _, p := range cs.Predicates {
		if p.Field == "body://name" {
			if p.Predicate("not xml at all <<<") {
				t.Error("should not match invalid XML")
			}
			return
		}
	}
	t.Error("body predicate not found")
}

func TestCompiler_XPathMissingNode(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "xpath-missing",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/test",
			Body: &scenario.BodyClause{
				ContentType: "xml",
				Conditions: []scenario.BodyCondition{
					{Extractor: "//nonexistent", Matcher: scenario.StringMatcher{Exact: "val"}},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	for _, p := range cs.Predicates {
		if p.Field == "body://nonexistent" {
			if p.Predicate(`<root><name>test</name></root>`) {
				t.Error("should not match when node is missing")
			}
			return
		}
	}
	t.Error("body predicate not found")
}

func TestCompiler_BodyFileResolution(t *testing.T) {
	dir := t.TempDir()
	bodyContent := `{"response":"from file"}`
	if err := os.WriteFile(filepath.Join(dir, "response.json"), []byte(bodyContent), 0o644); err != nil {
		t.Fatal(err)
	}

	compiler, err := services.NewCompiler(dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	s := &scenario.Scenario{
		ID: "body-file",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
		},
		Response: scenario.Response{
			Status:   200,
			BodyFile: "response.json",
		},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	if string(cs.Response.Body) != bodyContent {
		t.Errorf("expected body %q, got %q", bodyContent, cs.Response.Body)
	}
}

func TestCompiler_BodyFileAbsolutePathRejected(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "abs-path",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
		},
		Response: scenario.Response{
			Status:   200,
			BodyFile: "/etc/passwd",
		},
	}

	_, err := compiler.CompileScenario(s)
	if err == nil {
		t.Error("expected error for absolute body_file path")
	}
}

func TestCompiler_BodyFileTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	compiler, err := services.NewCompiler(dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	s := &scenario.Scenario{
		ID: "traversal",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
		},
		Response: scenario.Response{
			Status:   200,
			BodyFile: "../../etc/passwd",
		},
	}

	_, err = compiler.CompileScenario(s)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestCompiler_BodyFileMissing(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "missing-file",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
		},
		Response: scenario.Response{
			Status:   200,
			BodyFile: "nonexistent.json",
		},
	}

	_, err := compiler.CompileScenario(s)
	if err == nil {
		t.Error("expected error for missing body_file")
	}
}

// fakeRegistry implements TemplateRegistry for testing.
type fakeRegistry struct {
	err error
}

func (f *fakeRegistry) Compile(engine, name, source string) (match.BodyRenderer, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &fakeRenderer{body: []byte(source)}, nil
}

type fakeRenderer struct {
	body []byte
}

func (f *fakeRenderer) Render(_ match.RenderContext) ([]byte, error) {
	return f.body, nil
}

func TestCompiler_TemplateEngineNoRegistry(t *testing.T) {
	compiler := newTestCompiler(t) // nil registry

	s := &scenario.Scenario{
		ID: "template-no-registry",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
		},
		Response: scenario.Response{
			Status: 200,
			Body:   "hello ${name}",
			Engine: "expr",
		},
	}

	_, err := compiler.CompileScenario(s)
	if err == nil {
		t.Error("expected error when engine set but no registry")
	}
}

func TestCompiler_TemplateCompileError(t *testing.T) {
	dir := t.TempDir()
	reg := &fakeRegistry{err: fmt.Errorf("compile error")}
	compiler, err := services.NewCompiler(dir, reg)
	if err != nil {
		t.Fatal(err)
	}

	s := &scenario.Scenario{
		ID: "template-error",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
		},
		Response: scenario.Response{
			Status: 200,
			Body:   "bad template",
			Engine: "expr",
		},
	}

	_, err = compiler.CompileScenario(s)
	if err == nil {
		t.Error("expected error for template compilation failure")
	}
}

func TestCompiler_TemplateSuccess(t *testing.T) {
	dir := t.TempDir()
	reg := &fakeRegistry{}
	compiler, err := services.NewCompiler(dir, reg)
	if err != nil {
		t.Fatal(err)
	}

	s := &scenario.Scenario{
		ID: "template-ok",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
		},
		Response: scenario.Response{
			Status: 200,
			Body:   "hello world",
			Engine: "expr",
		},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	if cs.Response.Renderer == nil {
		t.Error("expected renderer to be set")
	}
}

func TestCompiler_PolicyWithPagination(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "with-pagination",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
		},
		Response: scenario.Response{Status: 200},
		Policy: &scenario.Policy{
			Pagination: &scenario.Pagination{
				Style:       "offset_limit",
				DefaultSize: 20,
				MaxSize:     50,
				DataPath:    "$.results",
				OffsetParam: "start",
				LimitParam:  "count",
				Envelope: scenario.PaginationEnvelope{
					DataField:        "items",
					TotalItemsField:  "total",
					TotalPagesField:  "pages",
					PageField:        "current_page",
					SizeField:        "per_page",
					HasNextField:     "more",
					HasPreviousField: "less",
				},
			},
		},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	if cs.Policy == nil || cs.Policy.Pagination == nil {
		t.Fatal("expected pagination policy")
	}

	p := cs.Policy.Pagination
	if p.Style != "offset_limit" {
		t.Errorf("expected offset_limit style, got %q", p.Style)
	}
	if p.DefaultSize != 20 {
		t.Errorf("expected default_size 20, got %d", p.DefaultSize)
	}
	if p.Envelope.DataField != "items" {
		t.Errorf("expected data field 'items', got %q", p.Envelope.DataField)
	}
}

func TestCompiler_BodyConditionInvalidRegex(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "bad-body-regex",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/test",
			Body: &scenario.BodyClause{
				ContentType: "json",
				Conditions: []scenario.BodyCondition{
					{Extractor: "$.name", Matcher: scenario.StringMatcher{Pattern: "[invalid"}},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	_, err := compiler.CompileScenario(s)
	if err == nil {
		t.Error("expected error for invalid regex in body condition")
	}
}

func TestPaginate_RootNonArray(t *testing.T) {
	body := []byte(`{"not":"an array"}`)
	cfg := &match.CompiledPagination{
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
	}

	_, err := services.Paginate(body, cfg, map[string]string{})
	if err == nil {
		t.Error("expected error for root non-array")
	}
}

func TestPaginate_JSONPathExtractionError(t *testing.T) {
	body := []byte(`{"items": [1,2,3]}`)
	cfg := &match.CompiledPagination{
		Style:       "page_size",
		PageParam:   "page",
		SizeParam:   "size",
		DefaultSize: 10,
		MaxSize:     100,
		DataPath:    "$.nonexistent.deep.path",
		Envelope: match.CompiledPaginationEnvelope{
			DataField:        "data",
			PageField:        "page",
			SizeField:        "size",
			TotalItemsField:  "total_items",
			TotalPagesField:  "total_pages",
			HasNextField:     "has_next",
			HasPreviousField: "has_previous",
		},
	}

	_, err := services.Paginate(body, cfg, map[string]string{})
	if err == nil {
		t.Error("expected error for invalid data path")
	}
}

func TestPaginate_NonArrayAtDataPath(t *testing.T) {
	body := []byte(`{"items": "not array"}`)
	cfg := &match.CompiledPagination{
		Style:       "page_size",
		PageParam:   "page",
		SizeParam:   "size",
		DefaultSize: 10,
		MaxSize:     100,
		DataPath:    "$.items",
		Envelope: match.CompiledPaginationEnvelope{
			DataField:        "data",
			PageField:        "page",
			SizeField:        "size",
			TotalItemsField:  "total_items",
			TotalPagesField:  "total_pages",
			HasNextField:     "has_next",
			HasPreviousField: "has_previous",
		},
	}

	_, err := services.Paginate(body, cfg, map[string]string{})
	if err == nil {
		t.Error("expected error for non-array at data path")
	}
}

func TestPaginate_OffsetLimitInvalidParams(t *testing.T) {
	body := []byte(`{"items": [1,2,3,4,5]}`)
	cfg := &match.CompiledPagination{
		Style:       "offset_limit",
		OffsetParam: "offset",
		LimitParam:  "limit",
		DefaultSize: 10,
		MaxSize:     100,
		DataPath:    "$.items",
		Envelope: match.CompiledPaginationEnvelope{
			DataField:        "data",
			PageField:        "page",
			SizeField:        "size",
			TotalItemsField:  "total_items",
			TotalPagesField:  "total_pages",
			HasNextField:     "has_next",
			HasPreviousField: "has_previous",
		},
	}

	// negative offset and non-numeric limit should use defaults
	result, err := services.Paginate(body, cfg, map[string]string{"offset": "-5", "limit": "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatal(err)
	}
	// offset defaults to 0, limit defaults to 10
	if env["has_previous"] != false {
		t.Error("expected has_previous=false with default offset")
	}
}

func TestPaginate_ZeroMaxSize(t *testing.T) {
	body := []byte(`{"items": [1,2,3]}`)
	cfg := &match.CompiledPagination{
		Style:       "page_size",
		PageParam:   "page",
		SizeParam:   "size",
		DefaultSize: 5,
		MaxSize:     0, // limit capped to 0, then fallback to 10
		DataPath:    "$.items",
		Envelope: match.CompiledPaginationEnvelope{
			DataField:        "data",
			PageField:        "page",
			SizeField:        "size",
			TotalItemsField:  "total_items",
			TotalPagesField:  "total_pages",
			HasNextField:     "has_next",
			HasPreviousField: "has_previous",
		},
	}

	result, err := services.Paginate(body, cfg, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatal(err)
	}
	// limit fallback to 10
	if env["size"].(float64) != 10 {
		t.Errorf("expected size 10 (fallback), got %v", env["size"])
	}
}

func TestCompiler_AllChildCompileError(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "all-error",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/test",
			Body: &scenario.BodyClause{
				ContentType: "json",
				All: []scenario.BodyClause{
					{
						ContentType: "json",
						Conditions: []scenario.BodyCondition{
							{Extractor: "$.name", Matcher: scenario.StringMatcher{Pattern: "[invalid"}},
						},
					},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	_, err := compiler.CompileScenario(s)
	if err == nil {
		t.Error("expected error for invalid regex in all combinator child")
	}
}

func TestCompiler_AnyChildCompileError(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "any-error",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/test",
			Body: &scenario.BodyClause{
				ContentType: "json",
				Any: []scenario.BodyClause{
					{
						ContentType: "json",
						Conditions: []scenario.BodyCondition{
							{Extractor: "$.name", Matcher: scenario.StringMatcher{Pattern: "[invalid"}},
						},
					},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	_, err := compiler.CompileScenario(s)
	if err == nil {
		t.Error("expected error for invalid regex in any combinator child")
	}
}

func TestCompiler_NotChildCompileError(t *testing.T) {
	compiler := newTestCompiler(t)

	s := &scenario.Scenario{
		ID: "not-error",
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/test",
			Body: &scenario.BodyClause{
				ContentType: "json",
				Not: &scenario.BodyClause{
					ContentType: "json",
					Conditions: []scenario.BodyCondition{
						{Extractor: "$.name", Matcher: scenario.StringMatcher{Pattern: "[invalid"}},
					},
				},
			},
		},
		Response: scenario.Response{Status: 200},
	}

	_, err := compiler.CompileScenario(s)
	if err == nil {
		t.Error("expected error for invalid regex in not combinator child")
	}
}

func TestCompiler_BodyFileWithTemplate(t *testing.T) {
	dir := t.TempDir()
	bodyContent := `Hello ${name}`
	if err := os.WriteFile(filepath.Join(dir, "template.txt"), []byte(bodyContent), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := &fakeRegistry{}
	compiler, err := services.NewCompiler(dir, reg)
	if err != nil {
		t.Fatal(err)
	}

	s := &scenario.Scenario{
		ID: "bodyfile-template",
		When: scenario.WhenClause{
			Method: "GET",
			Path:   "/test",
		},
		Response: scenario.Response{
			Status:   200,
			BodyFile: "template.txt",
			Engine:   "expr",
		},
	}

	cs, err := compiler.CompileScenario(s)
	if err != nil {
		t.Fatalf("CompileScenario failed: %v", err)
	}

	if cs.Response.Renderer == nil {
		t.Error("expected renderer for body_file + engine")
	}
}
