package services_test

import (
	"testing"

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
