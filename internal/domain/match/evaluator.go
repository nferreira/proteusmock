package match

import (
	"strings"

	"github.com/sophialabs/proteusmock/internal/domain/trace"
)

// IncomingRequest represents an HTTP request in domain terms, free of net/http.
type IncomingRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    []byte
}

// EvalResult holds the outcome of evaluating candidates against a request.
type EvalResult struct {
	Matched    *CompiledScenario
	Candidates []trace.CandidateResult
}

// Evaluator evaluates incoming requests against compiled scenarios.
type Evaluator struct{}

// NewEvaluator creates a new Evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate runs all candidates against the request and returns the best match.
// Candidates are assumed to be pre-sorted by priority descending, then ID ascending
// (as done by ScenarioIndex.Build).
func (e *Evaluator) Evaluate(req *IncomingRequest, candidates []*CompiledScenario) EvalResult {
	result := EvalResult{
		Candidates: make([]trace.CandidateResult, 0, len(candidates)),
	}

	// Build field value map for predicate evaluation.
	fieldValues := buildFieldValues(req)

	bodyStr := string(req.Body)

	for _, cs := range candidates {
		cr := trace.CandidateResult{
			ScenarioID:   cs.ID,
			ScenarioName: cs.Name,
			Matched:      true,
		}

		for _, fp := range cs.Predicates {
			val := resolveFieldValue(fp.Field, fieldValues, bodyStr)
			if !fp.Predicate(val) {
				cr.Matched = false
				cr.FailedField = fp.Field
				cr.FailedReason = "value did not match: " + val
				break
			}
		}

		result.Candidates = append(result.Candidates, cr)

		if cr.Matched && result.Matched == nil {
			result.Matched = cs
		}
	}

	return result
}

// resolveFieldValue returns the value for a field.
// Body predicates (field starting with "body:") receive the raw body
// since they internally parse and extract values.
func resolveFieldValue(field string, fieldValues map[string]string, body string) string {
	if strings.HasPrefix(field, "body:") || field == "body" {
		return body
	}
	return fieldValues[field]
}

func buildFieldValues(req *IncomingRequest) map[string]string {
	values := map[string]string{
		"method": req.Method,
		"path":   req.Path,
	}
	for k, v := range req.Headers {
		values["header:"+k] = v
	}
	return values
}
