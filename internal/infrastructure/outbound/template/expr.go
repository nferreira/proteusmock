package template

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

// ExprCompiler compiles body templates using the Expr language with ${ } interpolation.
type ExprCompiler struct{}

// Compile parses the source for ${ } delimiters and compiles each expression.
func (c *ExprCompiler) Compile(name, source string) (match.BodyRenderer, error) {
	segments, err := parseExprSegments(source)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expr template %q: %w", name, err)
	}

	// If no dynamic segments found, return a static renderer.
	hasDynamic := false
	for _, seg := range segments {
		if seg.program != nil {
			hasDynamic = true
			break
		}
	}
	if !hasDynamic {
		return &staticRenderer{body: []byte(source)}, nil
	}

	return &exprRenderer{segments: segments}, nil
}

type exprSegment struct {
	static  string
	program *vm.Program
}

func parseExprSegments(source string) ([]exprSegment, error) {
	var segments []exprSegment
	remaining := source

	for {
		idx := strings.Index(remaining, "${")
		if idx < 0 {
			if remaining != "" {
				segments = append(segments, exprSegment{static: remaining})
			}
			break
		}

		// Add static part before ${.
		if idx > 0 {
			segments = append(segments, exprSegment{static: remaining[:idx]})
		}

		// Find closing }.
		rest := remaining[idx+2:]
		closeIdx := findClosingBrace(rest)
		if closeIdx < 0 {
			return nil, fmt.Errorf("unclosed ${ at position %d", idx)
		}

		expression := rest[:closeIdx]
		program, err := expr.Compile(expression, expr.Env(exprEnv{}))
		if err != nil {
			return nil, fmt.Errorf("failed to compile expression %q: %w", expression, err)
		}
		segments = append(segments, exprSegment{program: program})
		remaining = rest[closeIdx+1:]
	}

	return segments, nil
}

// findClosingBrace finds the matching } accounting for nested braces.
func findClosingBrace(s string) int {
	depth := 0
	inString := false
	var stringChar byte
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			if ch == '\\' && i+1 < len(s) {
				i++ // skip escaped char
				continue
			}
			if ch == stringChar {
				inString = false
			}
			continue
		}
		switch ch {
		case '\'', '"':
			inString = true
			stringChar = ch
		case '{':
			depth++
		case '}':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

// exprEnv defines the environment available to Expr expressions.
type exprEnv struct {
	PathParam  func(string) string  `expr:"pathParam"`
	QueryParam func(string) string  `expr:"queryParam"`
	Header     func(string) string  `expr:"header"`
	Body       func() string        `expr:"body"`
	Now        func() string        `expr:"now"`
	NowFormat  func(string) string  `expr:"nowFormat"`
	UUID       func() string        `expr:"uuid"`
	RandomInt  func(int, int) int   `expr:"randomInt"`
	Seq        func(int, int) []int `expr:"seq"`
	ToJSON     func(any) string     `expr:"toJSON"`
	JsonPath   func(string) string  `expr:"jsonPath"`
}

type exprRenderer struct {
	segments []exprSegment
}

func (r *exprRenderer) Render(ctx match.RenderContext) ([]byte, error) {
	env := buildExprEnv(ctx)

	var buf strings.Builder
	for _, seg := range r.segments {
		if seg.program == nil {
			buf.WriteString(seg.static)
			continue
		}
		result, err := expr.Run(seg.program, env)
		if err != nil {
			return nil, fmt.Errorf("expression evaluation failed: %w", err)
		}
		fmt.Fprintf(&buf, "%v", result)
	}
	return []byte(buf.String()), nil
}

// staticRenderer returns a fixed body (used when no dynamic segments are found).
type staticRenderer struct {
	body []byte
}

func (r *staticRenderer) Render(match.RenderContext) ([]byte, error) {
	return r.body, nil
}
