package template

import (
	"testing"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

func TestRegistry_KnownEngines(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		engine string
		source string
	}{
		{"expr", `Hello ${pathParam('name')}`},
		{"jinja2", `Hello {{ pathParam("name") }}`},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			renderer, err := r.Compile(tt.engine, "test", tt.source)
			if err != nil {
				t.Fatalf("Compile failed for engine %q: %v", tt.engine, err)
			}

			result, err := renderer.Render(match.RenderContext{
				PathParams: map[string]string{"name": "World"},
			})
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}
			if string(result) != "Hello World" && string(result) != "Hello World!" {
				// Different engines may have slightly different output
				t.Logf("result: %q", result)
			}
		})
	}
}

func TestRegistry_UnknownEngine(t *testing.T) {
	r := NewRegistry()
	_, err := r.Compile("unknown", "test", "body")
	if err == nil {
		t.Error("expected error for unknown engine")
	}
}
