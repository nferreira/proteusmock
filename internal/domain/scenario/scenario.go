package scenario

// Scenario represents a single mock scenario definition.
type Scenario struct {
	ID       string
	Name     string
	Priority int
	When     WhenClause
	Response Response
	// Sequence, when non-empty, replaces Response: the Nth call to the scenario is
	// answered by the step whose cumulative Times covers N; the last step repeats.
	Sequence []SequenceStep
	// Bump names the counters this scenario increments each time it is served, so one request can change
	// what a later one is answered with (a write that a read then sees).
	Bump []string
	// Staged, when set, answers by the value of a counter instead of by call order.
	Staged *Staged
	Policy *Policy

	// SourceFile is the absolute path to the YAML file this scenario was loaded from.
	SourceFile string
	// SourceIndex is the index within a multi-scenario YAML file (0-based).
	// For single-scenario files, this is -1.
	SourceIndex int
}

// WhenClause defines the conditions for matching an incoming request.
type WhenClause struct {
	Method  string
	Path    string
	Headers map[string]StringMatcher
	Body    *BodyClause
}

// BodyClause represents conditions on the request body.
type BodyClause struct {
	ContentType string
	Conditions  []BodyCondition
	All         []BodyClause
	Any         []BodyClause
	Not         *BodyClause
}

// BodyCondition represents a single body extraction + matching rule.
type BodyCondition struct {
	// Extractor is a JSONPath or XPath expression.
	Extractor string
	// Matcher is the string matcher applied to the extracted value.
	Matcher StringMatcher
}

// StringMatcher represents a string matching rule.
// If Exact is non-empty, it's an exact match (prefixed with "=" in YAML).
// Otherwise, Pattern is treated as a regex.
type StringMatcher struct {
	Exact   string
	Pattern string
}

// IsExact returns true if this matcher uses exact comparison.
func (m StringMatcher) IsExact() bool {
	return m.Exact != ""
}

// Value returns the raw string value to match against.
func (m StringMatcher) Value() string {
	if m.Exact != "" {
		return m.Exact
	}
	return m.Pattern
}

// Response defines what the mock server returns.
type Response struct {
	Status      int
	Headers     map[string]string
	Body        string
	BodyFile    string
	ContentType string
	Engine      string // "" = static, "expr", "jinja2"
	FakerSeed   *int64
}

// Staged selects the response by the current value of a named counter: the stage with the highest
// From that is not above the value is used, so a stage keeps answering until a later one is reached.
type Staged struct {
	Counter string
	Stages  []Stage
}

// Stage is one answer of a Staged scenario, used from counter value From.
type Stage struct {
	From     int
	Response Response
}

// SequenceStep is one answer in a scenario's call-order sequence.
type SequenceStep struct {
	// Times is how many consecutive calls this step answers (minimum 1).
	Times    int
	Response Response
}

// Policy defines rate limiting, latency simulation, and pagination.
type Policy struct {
	RateLimit  *RateLimit
	Latency    *Latency
	Pagination *Pagination
}

// RateLimit configures token-bucket rate limiting.
type RateLimit struct {
	Rate  float64
	Burst int
	Key   string
}

// Latency configures response delay simulation.
type Latency struct {
	FixedMs  int
	JitterMs int
}

// PaginationStyle determines how pagination parameters are interpreted.
type PaginationStyle string

const (
	PaginationPageSize    PaginationStyle = "page_size"
	PaginationOffsetLimit PaginationStyle = "offset_limit"
)

// Pagination configures automatic response pagination.
type Pagination struct {
	Style       PaginationStyle
	PageParam   string
	SizeParam   string
	OffsetParam string
	LimitParam  string
	DefaultSize int
	MaxSize     int
	DataPath    string
	Envelope    PaginationEnvelope
}

// PaginationEnvelope configures the field names in the paginated response wrapper.
type PaginationEnvelope struct {
	DataField        string
	PageField        string
	SizeField        string
	TotalItemsField  string
	TotalPagesField  string
	HasNextField     string
	HasPreviousField string
}
