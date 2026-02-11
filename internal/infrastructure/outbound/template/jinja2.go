package template

import (
	"fmt"
	"time"

	"github.com/flosch/pongo2/v6"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

// Jinja2Compiler compiles body templates using Pongo2 (Django/Jinja2-style).
type Jinja2Compiler struct{}

// Compile parses the source as a Pongo2 template.
func (c *Jinja2Compiler) Compile(name, source string) (match.BodyRenderer, error) {
	tpl, err := pongo2.FromString(source)
	if err != nil {
		return nil, fmt.Errorf("failed to compile jinja2 template %q: %w", name, err)
	}
	return &jinja2Renderer{tpl: tpl}, nil
}

type jinja2Renderer struct {
	tpl *pongo2.Template
}

func (r *jinja2Renderer) Render(ctx match.RenderContext) ([]byte, error) {
	pongoCtx := pongo2.Context{
		"method":      ctx.Method,
		"path":        ctx.Path,
		"headers":     ctx.Headers,
		"queryParams": ctx.QueryParams,
		"pathParams":  ctx.PathParams,
		"body":        string(ctx.Body),
		"now":         ctx.Now,

		// Helper functions.
		"pathParam":  pongo2PathParam(ctx),
		"queryParam": pongo2QueryParam(ctx),
		"header":     pongo2Header(ctx),
		"uuid":       generateUUID,
		"randomInt": func(min, max int) int {
			if min >= max {
				return min
			}
			return min + randIntN(max-min+1)
		},
		"seq": func(start, end int) []int {
			return seqInts(start, end)
		},
		"toJSON": func(v any) string {
			return toJSONString(v)
		},
		"jsonPath": func(expression string) string {
			return extractJSONPath(ctx.Body, expression)
		},
		"nowFormat": func(layout string) string {
			t, err := time.Parse(time.RFC3339, ctx.Now)
			if err != nil {
				return ctx.Now
			}
			return t.Format(layout)
		},
	}

	result, err := r.tpl.Execute(pongoCtx)
	if err != nil {
		return nil, fmt.Errorf("jinja2 template render failed: %w", err)
	}
	return []byte(result), nil
}
