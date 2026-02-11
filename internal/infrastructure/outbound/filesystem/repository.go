package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sophialabs/proteusmock/internal/domain/scenario"
)

var _ scenario.Repository = (*YAMLRepository)(nil)

// YAMLRepository loads scenarios from YAML files in a directory tree.
type YAMLRepository struct {
	rootDir  string
	resolver *IncludeResolver
}

// NewYAMLRepository creates a repository rooted at rootDir.
func NewYAMLRepository(rootDir string) (*YAMLRepository, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve root directory: %w", err)
	}
	return &YAMLRepository{
		rootDir:  absRoot,
		resolver: NewIncludeResolver(absRoot),
	}, nil
}

// LoadAll walks the root directory for .yaml files and returns parsed scenarios.
func (r *YAMLRepository) LoadAll(_ context.Context) ([]*scenario.Scenario, error) {
	var scenarios []*scenario.Scenario

	err := filepath.WalkDir(r.rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		loaded, err := r.loadFile(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}
		scenarios = append(scenarios, loaded...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk scenarios directory: %w", err)
	}

	return scenarios, nil
}

func (r *YAMLRepository) loadFile(path string) ([]*scenario.Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse into yaml.Node tree to handle !include tags.
	var rootNode yaml.Node
	if err := yaml.Unmarshal(data, &rootNode); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	fileDir := filepath.Dir(path)
	if err := r.resolver.ResolveIncludes(&rootNode, fileDir); err != nil {
		return nil, fmt.Errorf("failed to resolve includes: %w", err)
	}

	// Decode resolved node tree into typed structures.
	// Support both single scenario and list of scenarios.
	var scenarios []*scenario.Scenario

	// Try as a list first.
	if rootNode.Kind == yaml.DocumentNode && len(rootNode.Content) > 0 {
		content := rootNode.Content[0]
		if content.Kind == yaml.SequenceNode {
			for _, item := range content.Content {
				s, err := decodeScenarioNode(item)
				if err != nil {
					return nil, err
				}
				scenarios = append(scenarios, s)
			}
			return scenarios, nil
		}

		// Single scenario.
		s, err := decodeScenarioNode(content)
		if err != nil {
			return nil, err
		}
		return []*scenario.Scenario{s}, nil
	}

	return nil, fmt.Errorf("unexpected YAML structure in %s", path)
}

func decodeScenarioNode(node *yaml.Node) (*scenario.Scenario, error) {
	var ys yamlScenario
	if err := node.Decode(&ys); err != nil {
		return nil, fmt.Errorf("failed to decode scenario: %w", err)
	}
	return toScenario(&ys), nil
}

func toScenario(ys *yamlScenario) *scenario.Scenario {
	s := &scenario.Scenario{
		ID:       ys.ID,
		Name:     ys.Name,
		Priority: ys.Priority,
		When: scenario.WhenClause{
			Method: ys.When.Method,
			Path:   ys.When.Path,
		},
		Response: scenario.Response{
			Status:      ys.Response.Status,
			Headers:     ys.Response.Headers,
			Body:        ys.Response.Body,
			BodyFile:    ys.Response.BodyFile,
			ContentType: ys.Response.ContentType,
			Engine:      ys.Response.Engine,
		},
	}

	if ys.When.Headers != nil {
		s.When.Headers = make(map[string]scenario.StringMatcher, len(ys.When.Headers))
		for k, v := range ys.When.Headers {
			s.When.Headers[k] = parseStringMatcher(v)
		}
	}

	if ys.When.Body != nil {
		s.When.Body = toBodyClause(ys.When.Body)
	}

	if ys.Policy != nil {
		s.Policy = toPolicy(ys.Policy)
	}

	return s
}

func parseStringMatcher(raw string) scenario.StringMatcher {
	if strings.HasPrefix(raw, "=") {
		return scenario.StringMatcher{Exact: raw[1:]}
	}
	return scenario.StringMatcher{Pattern: raw}
}

func toBodyClause(yb *yamlBody) *scenario.BodyClause {
	if yb == nil {
		return nil
	}

	bc := &scenario.BodyClause{
		ContentType: yb.ContentType,
	}

	for _, c := range yb.Conditions {
		bc.Conditions = append(bc.Conditions, scenario.BodyCondition{
			Extractor: c.Extractor,
			Matcher:   parseStringMatcher(c.Matcher),
		})
	}

	for _, a := range yb.All {
		allClause := toBodyClause(&a)
		if allClause != nil {
			bc.All = append(bc.All, *allClause)
		}
	}

	for _, a := range yb.Any {
		anyClause := toBodyClause(&a)
		if anyClause != nil {
			bc.Any = append(bc.Any, *anyClause)
		}
	}

	bc.Not = toBodyClause(yb.Not)

	return bc
}

func toPolicy(yp *yamlPolicy) *scenario.Policy {
	if yp == nil {
		return nil
	}

	p := &scenario.Policy{}

	if yp.RateLimit != nil {
		p.RateLimit = &scenario.RateLimit{
			Rate:  yp.RateLimit.Rate,
			Burst: yp.RateLimit.Burst,
			Key:   yp.RateLimit.Key,
		}
	}

	if yp.Latency != nil {
		p.Latency = &scenario.Latency{
			FixedMs:  yp.Latency.FixedMs,
			JitterMs: yp.Latency.JitterMs,
		}
	}

	if yp.Pagination != nil {
		p.Pagination = toPagination(yp.Pagination)
	}

	return p
}

func toPagination(yp *yamlPagination) *scenario.Pagination {
	p := &scenario.Pagination{
		Style:       scenario.PaginationStyle(yp.Style),
		PageParam:   yp.PageParam,
		SizeParam:   yp.SizeParam,
		OffsetParam: yp.OffsetParam,
		LimitParam:  yp.LimitParam,
		DefaultSize: yp.DefaultSize,
		MaxSize:     yp.MaxSize,
		DataPath:    yp.DataPath,
	}

	switch p.Style {
	case scenario.PaginationPageSize, scenario.PaginationOffsetLimit:
		// valid
	default:
		p.Style = scenario.PaginationPageSize
	}
	if p.PageParam == "" {
		p.PageParam = "page"
	}
	if p.SizeParam == "" {
		p.SizeParam = "size"
	}
	if p.OffsetParam == "" {
		p.OffsetParam = "offset"
	}
	if p.LimitParam == "" {
		p.LimitParam = "limit"
	}
	if p.DefaultSize == 0 {
		p.DefaultSize = 10
	}
	if p.MaxSize == 0 {
		p.MaxSize = 100
	}
	if p.DataPath == "" {
		p.DataPath = "$"
	}

	p.Envelope = toPaginationEnvelope(yp.Envelope)
	return p
}

func toPaginationEnvelope(ye *yamlPaginationEnvelope) scenario.PaginationEnvelope {
	env := scenario.PaginationEnvelope{
		DataField:        "data",
		PageField:        "page",
		SizeField:        "size",
		TotalItemsField:  "total_items",
		TotalPagesField:  "total_pages",
		HasNextField:     "has_next",
		HasPreviousField: "has_previous",
	}
	if ye == nil {
		return env
	}
	if ye.DataField != "" {
		env.DataField = ye.DataField
	}
	if ye.PageField != "" {
		env.PageField = ye.PageField
	}
	if ye.SizeField != "" {
		env.SizeField = ye.SizeField
	}
	if ye.TotalItemsField != "" {
		env.TotalItemsField = ye.TotalItemsField
	}
	if ye.TotalPagesField != "" {
		env.TotalPagesField = ye.TotalPagesField
	}
	if ye.HasNextField != "" {
		env.HasNextField = ye.HasNextField
	}
	if ye.HasPreviousField != "" {
		env.HasPreviousField = ye.HasPreviousField
	}
	return env
}
