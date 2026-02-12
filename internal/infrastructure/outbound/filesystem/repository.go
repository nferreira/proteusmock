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
			for i, item := range content.Content {
				s, err := decodeScenarioNode(item)
				if err != nil {
					return nil, err
				}
				s.SourceFile = path
				s.SourceIndex = i
				scenarios = append(scenarios, s)
			}
			return scenarios, nil
		}

		// Single scenario.
		s, err := decodeScenarioNode(content)
		if err != nil {
			return nil, err
		}
		s.SourceFile = path
		s.SourceIndex = -1
		return []*scenario.Scenario{s}, nil
	}

	return nil, fmt.Errorf("unexpected YAML structure in %s", path)
}

// LoadByID loads a single scenario by its ID.
func (r *YAMLRepository) LoadByID(ctx context.Context, id string) (*scenario.Scenario, error) {
	all, err := r.LoadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load scenarios: %w", err)
	}
	for _, s := range all {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, scenario.ErrNotFound
}

// SaveScenario writes scenario YAML content to disk.
// For existing scenarios (SourceFile set), it updates the file.
// For new scenarios (SourceFile empty), it creates a new file.
func (r *YAMLRepository) SaveScenario(_ context.Context, s *scenario.Scenario, yamlContent []byte) error {
	// Validate the YAML parses correctly.
	var check yaml.Node
	if err := yaml.Unmarshal(yamlContent, &check); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	if s.SourceFile == "" {
		// New scenario — create file at rootDir/scenarios/<id>.yaml
		dir := filepath.Join(r.rootDir, "scenarios")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create scenarios directory: %w", err)
		}
		target := filepath.Join(dir, s.ID+".yaml")

		// Path traversal check.
		if err := r.validatePathWithinRoot(target); err != nil {
			return err
		}

		return atomicWriteFile(target, yamlContent)
	}

	// Path traversal check for existing files.
	if err := r.validatePathWithinRoot(s.SourceFile); err != nil {
		return err
	}

	if s.SourceIndex < 0 {
		// Single-scenario file — replace entire file.
		return atomicWriteFile(s.SourceFile, yamlContent)
	}

	// Multi-scenario file — replace the entry at SourceIndex.
	return r.replaceInSequence(s.SourceFile, s.SourceIndex, yamlContent)
}

// DeleteScenario removes a scenario from its source file.
func (r *YAMLRepository) DeleteScenario(_ context.Context, sourceFile string, sourceIndex int) error {
	if err := r.validatePathWithinRoot(sourceFile); err != nil {
		return err
	}

	if sourceIndex < 0 {
		// Single-scenario file — delete the file.
		if err := os.Remove(sourceFile); err != nil {
			return fmt.Errorf("failed to delete scenario file: %w", err)
		}
		return nil
	}

	// Multi-scenario file — remove the entry at sourceIndex.
	return r.removeFromSequence(sourceFile, sourceIndex)
}

// ReadSourceYAML reads the raw YAML content for a specific scenario.
func (r *YAMLRepository) ReadSourceYAML(_ context.Context, s *scenario.Scenario) ([]byte, error) {
	if s.SourceFile == "" {
		return nil, fmt.Errorf("scenario has no source file")
	}

	data, err := os.ReadFile(s.SourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}

	if s.SourceIndex < 0 {
		// Single-scenario file — return entire content.
		return data, nil
	}

	// Multi-scenario file — extract the specific entry.
	return r.extractFromSequence(data, s.SourceIndex)
}

// validatePathWithinRoot ensures a path resolves within the root directory.
func (r *YAMLRepository) validatePathWithinRoot(path string) error {
	resolved, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		// If the directory doesn't exist yet, check the absolute path.
		abs, absErr := filepath.Abs(path)
		if absErr != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
		if !strings.HasPrefix(abs, r.rootDir) {
			return fmt.Errorf("path traversal denied: %s is outside root %s", path, r.rootDir)
		}
		return nil
	}
	if !strings.HasPrefix(resolved, r.rootDir) {
		return fmt.Errorf("path traversal denied: %s is outside root %s", path, r.rootDir)
	}
	return nil
}

// atomicWriteFile writes content to a temp file then renames it to the target path.
func atomicWriteFile(target string, content []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".proteusmock-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	return nil
}

// replaceInSequence replaces an entry at a given index in a YAML sequence file.
func (r *YAMLRepository) replaceInSequence(filePath string, index int, newContent []byte) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var rootNode yaml.Node
	if err := yaml.Unmarshal(data, &rootNode); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	if rootNode.Kind != yaml.DocumentNode || len(rootNode.Content) == 0 {
		return fmt.Errorf("unexpected YAML structure")
	}
	seq := rootNode.Content[0]
	if seq.Kind != yaml.SequenceNode {
		return fmt.Errorf("file is not a YAML sequence")
	}
	if index >= len(seq.Content) {
		return fmt.Errorf("index %d out of range (file has %d entries)", index, len(seq.Content))
	}

	// Parse the new content into a node.
	var newNode yaml.Node
	if err := yaml.Unmarshal(newContent, &newNode); err != nil {
		return fmt.Errorf("failed to parse replacement YAML: %w", err)
	}
	if newNode.Kind != yaml.DocumentNode || len(newNode.Content) == 0 {
		return fmt.Errorf("unexpected replacement YAML structure")
	}

	seq.Content[index] = newNode.Content[0]

	out, err := yaml.Marshal(&rootNode)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return atomicWriteFile(filePath, out)
}

// removeFromSequence removes an entry at a given index from a YAML sequence file.
func (r *YAMLRepository) removeFromSequence(filePath string, index int) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var rootNode yaml.Node
	if err := yaml.Unmarshal(data, &rootNode); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	if rootNode.Kind != yaml.DocumentNode || len(rootNode.Content) == 0 {
		return fmt.Errorf("unexpected YAML structure")
	}
	seq := rootNode.Content[0]
	if seq.Kind != yaml.SequenceNode {
		return fmt.Errorf("file is not a YAML sequence")
	}
	if index >= len(seq.Content) {
		return fmt.Errorf("index %d out of range (file has %d entries)", index, len(seq.Content))
	}

	// Remove the entry.
	seq.Content = append(seq.Content[:index], seq.Content[index+1:]...)

	if len(seq.Content) == 0 {
		// No more entries — delete the file.
		return os.Remove(filePath)
	}

	out, err := yaml.Marshal(&rootNode)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return atomicWriteFile(filePath, out)
}

// extractFromSequence extracts a single entry from a YAML sequence.
func (r *YAMLRepository) extractFromSequence(data []byte, index int) ([]byte, error) {
	var rootNode yaml.Node
	if err := yaml.Unmarshal(data, &rootNode); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if rootNode.Kind != yaml.DocumentNode || len(rootNode.Content) == 0 {
		return nil, fmt.Errorf("unexpected YAML structure")
	}
	seq := rootNode.Content[0]
	if seq.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("file is not a YAML sequence")
	}
	if index >= len(seq.Content) {
		return nil, fmt.Errorf("index %d out of range (file has %d entries)", index, len(seq.Content))
	}

	out, err := yaml.Marshal(seq.Content[index])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entry: %w", err)
	}
	return out, nil
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
