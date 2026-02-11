package template

import (
	"fmt"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

// EngineCompiler compiles a template source string into a BodyRenderer.
type EngineCompiler interface {
	Compile(name, source string) (match.BodyRenderer, error)
}

// Registry maps engine names to their compilers.
type Registry struct {
	engines map[string]EngineCompiler
}

// NewRegistry creates a registry with the built-in engines (expr, jinja2).
func NewRegistry() *Registry {
	return &Registry{
		engines: map[string]EngineCompiler{
			"expr":   &ExprCompiler{},
			"jinja2": &Jinja2Compiler{},
		},
	}
}

// Compile resolves the engine by name and compiles the source.
func (r *Registry) Compile(engine, name, source string) (match.BodyRenderer, error) {
	ec, ok := r.engines[engine]
	if !ok {
		return nil, fmt.Errorf("unknown template engine: %q (supported: expr, jinja2)", engine)
	}
	return ec.Compile(name, source)
}
