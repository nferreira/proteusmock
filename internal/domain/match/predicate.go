package match

import "sync/atomic"

// Predicate tests a string value and returns true if it matches.
type Predicate func(string) bool

// And returns a predicate that requires all predicates to match.
func And(predicates ...Predicate) Predicate {
	return func(s string) bool {
		for _, p := range predicates {
			if !p(s) {
				return false
			}
		}
		return true
	}
}

// Or returns a predicate that requires at least one predicate to match.
func Or(predicates ...Predicate) Predicate {
	return func(s string) bool {
		for _, p := range predicates {
			if p(s) {
				return true
			}
		}
		return false
	}
}

// Not returns a predicate that inverts the given predicate.
func Not(p Predicate) Predicate {
	return func(s string) bool {
		return !p(s)
	}
}

// Always returns a predicate that always matches.
func Always() Predicate {
	return func(string) bool { return true }
}

// Never returns a predicate that never matches.
func Never() Predicate {
	return func(string) bool { return false }
}

// FieldPredicate binds a named field to its compiled predicate.
type FieldPredicate struct {
	Field     string
	Predicate Predicate
}

// CompiledScenario holds a scenario with its compiled field predicates.
type CompiledScenario struct {
	ID         string
	Name       string
	Priority   int
	Method     string
	PathKey    string
	Predicates []FieldPredicate
	Response   CompiledResponse
	Policy     *CompiledPolicy

	// Steps is the compiled call-order sequence; empty means Response always applies.
	Steps []CompiledStep
	calls atomic.Int64
}

// CompiledStep is one compiled step of a response sequence.
type CompiledStep struct {
	Times    int
	Response CompiledResponse
}

// NextResponse returns the response for this call. Without a sequence it is
// Response; with one, each call advances a counter through the steps and the
// last step repeats once the sequence is exhausted.
func (cs *CompiledScenario) NextResponse() CompiledResponse {
	if len(cs.Steps) == 0 {
		return cs.Response
	}
	n := cs.calls.Add(1) - 1
	for _, step := range cs.Steps {
		if n < int64(step.Times) {
			return step.Response
		}
		n -= int64(step.Times)
	}
	return cs.Steps[len(cs.Steps)-1].Response
}

// ResetSequence rewinds the sequence so the next call gets the first step.
func (cs *CompiledScenario) ResetSequence() { cs.calls.Store(0) }

// BodyRenderer renders a response body dynamically. Nil means static body.
type BodyRenderer interface {
	Render(ctx RenderContext) ([]byte, error)
}

// RenderContext provides request data for dynamic body rendering.
type RenderContext struct {
	Method      string
	Path        string
	Headers     map[string]string
	QueryParams map[string]string
	PathParams  map[string]string
	Body        []byte
	Now         string // ISO-8601 timestamp
	FakerSeed   *int64
}

// CompiledResponse is a resolved response ready to serve.
type CompiledResponse struct {
	Status      int
	Headers     map[string]string
	Body        []byte       // used when Renderer is nil
	Renderer    BodyRenderer // non-nil for dynamic bodies
	ContentType string
	FakerSeed   *int64
}

// CompiledPolicy holds resolved policy configuration.
type CompiledPolicy struct {
	RateLimit  *CompiledRateLimit
	Latency    *CompiledLatency
	Pagination *CompiledPagination
}

// CompiledRateLimit holds rate limit parameters.
type CompiledRateLimit struct {
	Rate  float64
	Burst int
	Key   string
}

// CompiledLatency holds latency simulation parameters.
type CompiledLatency struct {
	FixedMs  int
	JitterMs int
}

// CompiledPagination holds resolved pagination configuration.
type CompiledPagination struct {
	Style       string // "page_size" or "offset_limit"
	PageParam   string
	SizeParam   string
	OffsetParam string
	LimitParam  string
	DefaultSize int
	MaxSize     int
	DataPath    string
	Envelope    CompiledPaginationEnvelope
}

// CompiledPaginationEnvelope holds resolved envelope field names.
type CompiledPaginationEnvelope struct {
	DataField        string
	PageField        string
	SizeField        string
	TotalItemsField  string
	TotalPagesField  string
	HasNextField     string
	HasPreviousField string
}
