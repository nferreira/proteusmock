package template

import (
	"strings"
	"testing"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

func TestJinja2Compiler_SimpleVariable(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `Hello {{ pathParam("name") }}!`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		PathParams: map[string]string{"name": "World"},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "Hello World!" {
		t.Errorf("expected 'Hello World!', got %q", result)
	}
}

func TestJinja2Compiler_Conditional(t *testing.T) {
	c := &Jinja2Compiler{}
	source := `{% if header("X-Mode") == "debug" %}verbose{% else %}brief{% endif %}`
	renderer, err := c.Compile("test", source)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{"debug mode", map[string]string{"X-Mode": "debug"}, "verbose"},
		{"normal mode", map[string]string{"X-Mode": "prod"}, "brief"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.Render(match.RenderContext{Headers: tt.headers})
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}
			if string(result) != tt.want {
				t.Errorf("expected %q, got %q", tt.want, result)
			}
		})
	}
}

func TestJinja2Compiler_Loop(t *testing.T) {
	c := &Jinja2Compiler{}
	source := `{% for i in seq(1, 3) %}{{ i }}{% endfor %}`
	renderer, err := c.Compile("test", source)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "123" {
		t.Errorf("expected '123', got %q", result)
	}
}

func TestJinja2Compiler_StaticBody(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `plain text with no templates`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "plain text with no templates" {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestJinja2Compiler_InvalidSyntax(t *testing.T) {
	c := &Jinja2Compiler{}
	_, err := c.Compile("test", `{% if %}broken{% endif %}`)
	if err == nil {
		t.Error("expected compile error for invalid syntax")
	}
}

func TestJinja2Compiler_ContextVariables(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ method }} {{ path }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		Method: "POST",
		Path:   "/api/test",
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "POST /api/test" {
		t.Errorf("expected 'POST /api/test', got %q", result)
	}
}

func TestJinja2Compiler_HeaderCaseInsensitive(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ header("content-type") }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		Headers: map[string]string{"Content-Type": "application/json"},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "application/json" {
		t.Errorf("expected 'application/json', got %q", result)
	}
}

func TestJinja2Compiler_UUID(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ uuid() }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	s := string(result)
	if len(s) != 36 || s[8] != '-' || s[13] != '-' {
		t.Errorf("expected UUID format, got %q", s)
	}
}

func TestJinja2Compiler_Now(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ now }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		Now: "2025-01-15T10:30:00Z",
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "2025-01-15T10:30:00Z" {
		t.Errorf("expected timestamp, got %q", result)
	}
}

func TestJinja2Compiler_QueryParam(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `page={{ queryParam("page") }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		QueryParams: map[string]string{"page": "5"},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "page=5" {
		t.Errorf("expected 'page=5', got %q", result)
	}
}

func TestJinja2Compiler_Body(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `echo: {{ body }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		Body: []byte("hello"),
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "echo: hello" {
		t.Errorf("expected 'echo: hello', got %q", result)
	}
}

func TestJinja2Compiler_ToJSON(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ toJSON(seq(1, 3)) }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "[1,2,3]" {
		t.Errorf("expected '[1,2,3]', got %q", result)
	}
}

func TestJinja2Compiler_NowFormat(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ nowFormat("2006-01-02") }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		Now: "2025-01-15T10:30:00Z",
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "2025-01-15" {
		t.Errorf("expected '2025-01-15', got %q", result)
	}
}

func TestJinja2Compiler_NowFormatInvalidDate(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ nowFormat("2006-01-02") }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		Now: "not-a-date",
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "not-a-date" {
		t.Errorf("expected fallback 'not-a-date', got %q", result)
	}
}

func TestJinja2Compiler_RandomInt(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ randomInt(5, 5) }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "5" {
		t.Errorf("expected '5', got %q", result)
	}
}

func TestJinja2Compiler_HeaderMissing(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `[{{ header("X-Missing") }}]`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		Headers: map[string]string{},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "[]" {
		t.Errorf("expected '[]', got %q", result)
	}
}

func TestJinja2Compiler_PathParam(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `id={{ pathParam("id") }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		PathParams: map[string]string{"id": "42"},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "id=42" {
		t.Errorf("expected 'id=42', got %q", result)
	}
}

func TestJinja2Compiler_SeqEmpty(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ toJSON(seq(5, 3)) }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "null" {
		t.Errorf("expected 'null', got %q", result)
	}
}

func TestJinja2Compiler_JsonPath(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `name={{ jsonPath("$.name") }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		Body: []byte(`{"name":"Alice"}`),
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(string(result), "Alice") {
		t.Errorf("expected result to contain 'Alice', got %q", result)
	}
}

func TestJinja2Compiler_RandomIntRange(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ randomInt(1, 10) }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if len(string(result)) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestJinja2Compiler_JsonPathInvalidJSON(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `[{{ jsonPath("$.name") }}]`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		Body: []byte("not json"),
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "[]" {
		t.Errorf("expected '[]', got %q", result)
	}
}
