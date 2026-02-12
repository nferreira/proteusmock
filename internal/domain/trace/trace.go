package trace

import "time"

// Entry represents a single match trace entry.
type Entry struct {
	Timestamp   time.Time         `json:"timestamp"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	MatchedID   string            `json:"matched_id"`
	Candidates  []CandidateResult `json:"candidates"`
	RateLimited bool              `json:"rate_limited"`
}

// CandidateResult records the evaluation result for a single candidate scenario.
type CandidateResult struct {
	ScenarioID   string `json:"scenario_id"`
	ScenarioName string `json:"scenario_name"`
	Matched      bool   `json:"matched"`
	FailedField  string `json:"failed_field,omitempty"`
	FailedReason string `json:"failed_reason,omitempty"`
}
