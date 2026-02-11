package trace

import "time"

// Entry represents a single match trace entry.
type Entry struct {
	Timestamp   time.Time
	Method      string
	Path        string
	MatchedID   string
	Candidates  []CandidateResult
	RateLimited bool
}

// CandidateResult records the evaluation result for a single candidate scenario.
type CandidateResult struct {
	ScenarioID   string
	ScenarioName string
	Matched      bool
	FailedField  string
	FailedReason string
}
