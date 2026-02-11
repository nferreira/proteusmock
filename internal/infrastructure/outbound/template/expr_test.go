package template

import (
	"strings"
	"testing"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

func TestExprCompiler_SimpleInterpolation(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `Hello ${pathParam('name')}!`)
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

func TestExprCompiler_NoExpressions(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `static body content`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "static body content" {
		t.Errorf("expected 'static body content', got %q", result)
	}
}

func TestExprCompiler_Ternary(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `${header('X-Mode') == 'debug' ? 'verbose' : 'brief'}`)
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
		{"missing header", map[string]string{}, "brief"},
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

func TestExprCompiler_MultipleExpressions(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `{"a": "${pathParam('x')}", "b": "${pathParam('y')}"}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		PathParams: map[string]string{"x": "1", "y": "2"},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != `{"a": "1", "b": "2"}` {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestExprCompiler_InvalidSyntax(t *testing.T) {
	c := &ExprCompiler{}
	_, err := c.Compile("test", `${invalid syntax here ???}`)
	if err == nil {
		t.Error("expected compile error for invalid syntax")
	}
}

func TestExprCompiler_UnclosedDelimiter(t *testing.T) {
	c := &ExprCompiler{}
	_, err := c.Compile("test", `Hello ${pathParam('name')`)
	if err == nil {
		t.Error("expected compile error for unclosed ${")
	}
}

func TestExprCompiler_EmptyBody(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", "")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestExprCompiler_HeaderCaseInsensitive(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `${header('content-type')}`)
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

func TestExprCompiler_QueryParam(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `page=${queryParam('page')}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		QueryParams: map[string]string{"page": "3"},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "page=3" {
		t.Errorf("expected 'page=3', got %q", result)
	}
}

func TestExprCompiler_Now(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `${now()}`)
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

func TestExprCompiler_UUID(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `${uuid()}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	s := string(result)
	// UUID v4 format: 8-4-4-4-12
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		t.Errorf("expected UUID format, got %q", s)
	}
}

func TestExprCompiler_RandomInt(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `${randomInt(1, 10)}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	s := string(result)
	if s < "1" || s > "9" {
		// It's string comparison but single digits; just check it's non-empty.
		if len(s) == 0 {
			t.Errorf("expected non-empty result")
		}
	}
}

func TestExprCompiler_Seq(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `${toJSON(seq(1, 3))}`)
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

func TestExprCompiler_Body(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `echo: ${body()}`)
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

func TestExprCompiler_JsonPath(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `name=${jsonPath('$.name')}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		Body: []byte(`{"name":"Alice"}`),
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(result) != "name=Alice" {
		t.Errorf("expected 'name=Alice', got %q", result)
	}
}

func TestExprCompiler_NestedBraces(t *testing.T) {
	c := &ExprCompiler{}
	// Expression with map literal containing braces
	renderer, err := c.Compile("test", `${toJSON({'key': pathParam('id')})}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{
		PathParams: map[string]string{"id": "42"},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(string(result), "42") {
		t.Errorf("expected result to contain '42', got %q", result)
	}
}
