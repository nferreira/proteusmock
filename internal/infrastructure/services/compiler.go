package services

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/antchfx/xmlquery"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/domain/scenario"
)

// TemplateRegistry compiles template sources into body renderers by engine name.
type TemplateRegistry interface {
	Compile(engine, name, source string) (match.BodyRenderer, error)
}

// Compiler transforms domain scenarios into compiled scenarios with predicates.
type Compiler struct {
	rootDir  string
	registry TemplateRegistry // nil means no template support
}

// NewCompiler creates a new Compiler bound to the given root directory for body_file resolution.
// registry may be nil, in which case scenarios with an engine field will fail to compile.
func NewCompiler(rootDir string, registry TemplateRegistry) (*Compiler, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve root directory: %w", err)
	}
	return &Compiler{rootDir: absRoot, registry: registry}, nil
}

// CompileScenario turns a Scenario into a CompiledScenario.
func (c *Compiler) CompileScenario(s *scenario.Scenario) (*match.CompiledScenario, error) {
	predicates, err := c.compileWhen(&s.When)
	if err != nil {
		return nil, fmt.Errorf("failed to compile scenario %q: %w", s.ID, err)
	}

	resp, err := c.compileResponse(&s.Response)
	if err != nil {
		return nil, fmt.Errorf("failed to compile response for %q: %w", s.ID, err)
	}

	cs := &match.CompiledScenario{
		ID:         s.ID,
		Name:       s.Name,
		Priority:   s.Priority,
		Method:     s.When.Method,
		PathKey:    s.When.Method + ":" + s.When.Path,
		Predicates: predicates,
		Response:   resp,
	}

	if s.Policy != nil {
		cs.Policy = compilePolicy(s.Policy)
	}

	return cs, nil
}

func (c *Compiler) compileWhen(w *scenario.WhenClause) ([]match.FieldPredicate, error) {
	var predicates []match.FieldPredicate

	// Method predicate — always exact.
	if w.Method != "" {
		predicates = append(predicates, match.FieldPredicate{
			Field:     "method",
			Predicate: exactPredicate(w.Method),
		})
	}

	// Header predicates — sorted for deterministic ordering.
	headerNames := make([]string, 0, len(w.Headers))
	for name := range w.Headers {
		headerNames = append(headerNames, name)
	}
	sort.Strings(headerNames)

	for _, name := range headerNames {
		matcher := w.Headers[name]
		p, err := compileStringMatcher(matcher)
		if err != nil {
			return nil, fmt.Errorf("header %q: %w", name, err)
		}
		// Canonicalize header name to match HTTP canonical form.
		canonicalName := http.CanonicalHeaderKey(name)
		predicates = append(predicates, match.FieldPredicate{
			Field:     "header:" + canonicalName,
			Predicate: p,
		})
	}

	// Body predicates.
	if w.Body != nil {
		bodyPreds, err := c.compileBody(w.Body)
		if err != nil {
			return nil, err
		}
		predicates = append(predicates, bodyPreds...)
	}

	return predicates, nil
}

func (c *Compiler) compileBody(bc *scenario.BodyClause) ([]match.FieldPredicate, error) {
	var predicates []match.FieldPredicate

	for _, cond := range bc.Conditions {
		p, err := c.compileBodyCondition(cond, bc.ContentType)
		if err != nil {
			return nil, err
		}
		predicates = append(predicates, p)
	}

	// Boolean combinators.
	if len(bc.All) > 0 {
		var allPreds []match.Predicate
		for _, child := range bc.All {
			childPreds, err := c.compileBody(&child)
			if err != nil {
				return nil, err
			}
			for _, cp := range childPreds {
				allPreds = append(allPreds, cp.Predicate)
			}
		}
		predicates = append(predicates, match.FieldPredicate{
			Field:     "body:all",
			Predicate: match.And(allPreds...),
		})
	}

	if len(bc.Any) > 0 {
		var anyPreds []match.Predicate
		for _, child := range bc.Any {
			childPreds, err := c.compileBody(&child)
			if err != nil {
				return nil, err
			}
			for _, cp := range childPreds {
				anyPreds = append(anyPreds, cp.Predicate)
			}
		}
		predicates = append(predicates, match.FieldPredicate{
			Field:     "body:any",
			Predicate: match.Or(anyPreds...),
		})
	}

	if bc.Not != nil {
		notPreds, err := c.compileBody(bc.Not)
		if err != nil {
			return nil, err
		}
		if len(notPreds) > 0 {
			var inner []match.Predicate
			for _, np := range notPreds {
				inner = append(inner, np.Predicate)
			}
			predicates = append(predicates, match.FieldPredicate{
				Field:     "body:not",
				Predicate: match.Not(match.And(inner...)),
			})
		}
	}

	return predicates, nil
}

func (c *Compiler) compileBodyCondition(cond scenario.BodyCondition, contentType string) (match.FieldPredicate, error) {
	matcher, err := compileStringMatcher(cond.Matcher)
	if err != nil {
		return match.FieldPredicate{}, fmt.Errorf("body condition %q: %w", cond.Extractor, err)
	}

	fieldName := "body:" + cond.Extractor

	switch strings.ToLower(contentType) {
	case "json":
		return match.FieldPredicate{
			Field:     fieldName,
			Predicate: jsonPathPredicate(cond.Extractor, matcher),
		}, nil
	case "xml":
		return match.FieldPredicate{
			Field:     fieldName,
			Predicate: xpathPredicate(cond.Extractor, matcher),
		}, nil
	default:
		// No content type specified — match against raw body.
		return match.FieldPredicate{
			Field:     "body",
			Predicate: matcher,
		}, nil
	}
}

func compileStringMatcher(m scenario.StringMatcher) (match.Predicate, error) {
	if m.IsExact() {
		return exactPredicate(m.Exact), nil
	}
	if m.Pattern == "" {
		return match.Always(), nil
	}
	return regexPredicate(m.Pattern)
}

func exactPredicate(expected string) match.Predicate {
	return func(s string) bool {
		return s == expected
	}
}

func regexPredicate(pattern string) (match.Predicate, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}
	return func(s string) bool {
		return re.MatchString(s)
	}, nil
}

// jsonPathPredicate creates a predicate that extracts a value via JSONPath and matches it.
func jsonPathPredicate(expr string, valueMatcher match.Predicate) match.Predicate {
	return func(body string) bool {
		var data any
		if err := parseJSON(body, &data); err != nil {
			return false
		}

		result, err := jsonpath.Get(expr, data)
		if err != nil {
			return false
		}

		return valueMatcher(fmt.Sprintf("%v", result))
	}
}

func parseJSON(s string, v any) error {
	dec := strings.NewReader(s)
	return decodeJSON(dec, v)
}

// xpathPredicate creates a predicate that extracts a value via XPath and matches it.
func xpathPredicate(expr string, valueMatcher match.Predicate) match.Predicate {
	return func(body string) bool {
		doc, err := xmlquery.Parse(strings.NewReader(body))
		if err != nil {
			return false
		}

		node := xmlquery.FindOne(doc, expr)
		if node == nil {
			return false
		}

		return valueMatcher(node.InnerText())
	}
}

func (c *Compiler) compileResponse(r *scenario.Response) (match.CompiledResponse, error) {
	resp := match.CompiledResponse{
		Status:      r.Status,
		Headers:     r.Headers,
		ContentType: r.ContentType,
	}

	if resp.Status == 0 {
		resp.Status = 200
	}

	// Resolve body content (inline or from file).
	var bodySource string
	if r.BodyFile != "" {
		resolved, err := c.resolveBodyFilePath(r.BodyFile)
		if err != nil {
			return resp, err
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			return resp, fmt.Errorf("failed to read body_file %q: %w", r.BodyFile, err)
		}
		bodySource = string(data)
	} else {
		bodySource = r.Body
	}

	// If engine is set, compile as template; otherwise treat as static.
	if r.Engine != "" {
		if c.registry == nil {
			return resp, fmt.Errorf("template engine %q requested but no registry configured", r.Engine)
		}
		name := r.BodyFile
		if name == "" {
			name = "inline"
		}
		renderer, err := c.registry.Compile(r.Engine, name, bodySource)
		if err != nil {
			return resp, fmt.Errorf("failed to compile template (engine=%s): %w", r.Engine, err)
		}
		resp.Renderer = renderer
	} else {
		resp.Body = []byte(bodySource)
	}

	return resp, nil
}

// resolveBodyFilePath resolves and validates body_file paths to prevent directory traversal.
func (c *Compiler) resolveBodyFilePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute paths not allowed in body_file: %s", path)
	}

	resolved := filepath.Join(c.rootDir, path)

	// Evaluate symlinks and verify the path stays within rootDir.
	realPath, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		realPath = filepath.Clean(resolved)
	}

	absRoot, err := filepath.EvalSymlinks(c.rootDir)
	if err != nil {
		absRoot = c.rootDir
	}

	if !strings.HasPrefix(realPath, absRoot) {
		return "", fmt.Errorf("body_file path %q escapes root directory", path)
	}

	return resolved, nil
}

func compilePolicy(p *scenario.Policy) *match.CompiledPolicy {
	cp := &match.CompiledPolicy{}

	if p.RateLimit != nil {
		cp.RateLimit = &match.CompiledRateLimit{
			Rate:  p.RateLimit.Rate,
			Burst: p.RateLimit.Burst,
			Key:   p.RateLimit.Key,
		}
	}

	if p.Latency != nil {
		cp.Latency = &match.CompiledLatency{
			FixedMs:  p.Latency.FixedMs,
			JitterMs: p.Latency.JitterMs,
		}
	}

	if p.Pagination != nil {
		cp.Pagination = &match.CompiledPagination{
			Style:       string(p.Pagination.Style),
			PageParam:   p.Pagination.PageParam,
			SizeParam:   p.Pagination.SizeParam,
			OffsetParam: p.Pagination.OffsetParam,
			LimitParam:  p.Pagination.LimitParam,
			DefaultSize: p.Pagination.DefaultSize,
			MaxSize:     p.Pagination.MaxSize,
			DataPath:    p.Pagination.DataPath,
			Envelope: match.CompiledPaginationEnvelope{
				DataField:        p.Pagination.Envelope.DataField,
				PageField:        p.Pagination.Envelope.PageField,
				SizeField:        p.Pagination.Envelope.SizeField,
				TotalItemsField:  p.Pagination.Envelope.TotalItemsField,
				TotalPagesField:  p.Pagination.Envelope.TotalPagesField,
				HasNextField:     p.Pagination.Envelope.HasNextField,
				HasPreviousField: p.Pagination.Envelope.HasPreviousField,
			},
		}
	}

	return cp
}
