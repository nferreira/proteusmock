package match

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
}

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
}

// CompiledResponse is a resolved response ready to serve.
type CompiledResponse struct {
	Status      int
	Headers     map[string]string
	Body        []byte       // used when Renderer is nil
	Renderer    BodyRenderer // non-nil for dynamic bodies
	ContentType string
}

// CompiledPolicy holds resolved policy configuration.
type CompiledPolicy struct {
	RateLimit *CompiledRateLimit
	Latency   *CompiledLatency
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
