package scenario_test

import (
	"testing"

	"github.com/sophialabs/proteusmock/internal/domain/scenario"
)

func TestStringMatcher_IsExact(t *testing.T) {
	tests := []struct {
		name    string
		matcher scenario.StringMatcher
		want    bool
	}{
		{
			name:    "exact matcher",
			matcher: scenario.StringMatcher{Exact: "hello"},
			want:    true,
		},
		{
			name:    "regex matcher",
			matcher: scenario.StringMatcher{Pattern: "hello.*"},
			want:    false,
		},
		{
			name:    "empty matcher",
			matcher: scenario.StringMatcher{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.matcher.IsExact(); got != tt.want {
				t.Errorf("IsExact() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringMatcher_Value(t *testing.T) {
	tests := []struct {
		name    string
		matcher scenario.StringMatcher
		want    string
	}{
		{
			name:    "exact value",
			matcher: scenario.StringMatcher{Exact: "hello"},
			want:    "hello",
		},
		{
			name:    "pattern value",
			matcher: scenario.StringMatcher{Pattern: "hello.*"},
			want:    "hello.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.matcher.Value(); got != tt.want {
				t.Errorf("Value() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScenarioStructure(t *testing.T) {
	s := &scenario.Scenario{
		ID:       "test-1",
		Name:     "Test Scenario",
		Priority: 10,
		When: scenario.WhenClause{
			Method: "POST",
			Path:   "/api/v1/test",
			Headers: map[string]scenario.StringMatcher{
				"Content-Type": {Exact: "application/json"},
			},
			Body: &scenario.BodyClause{
				ContentType: "json",
				Conditions: []scenario.BodyCondition{
					{
						Extractor: "$.name",
						Matcher:   scenario.StringMatcher{Exact: "test"},
					},
				},
			},
		},
		Response: scenario.Response{
			Status: 200,
			Body:   `{"ok": true}`,
		},
		Policy: &scenario.Policy{
			RateLimit: &scenario.RateLimit{Rate: 10, Burst: 5, Key: "ip"},
			Latency:   &scenario.Latency{FixedMs: 100, JitterMs: 50},
		},
	}

	if s.ID != "test-1" {
		t.Errorf("unexpected ID: %s", s.ID)
	}
	if s.When.Method != "POST" {
		t.Errorf("unexpected method: %s", s.When.Method)
	}
	if s.When.Body == nil {
		t.Fatal("expected body clause")
	}
	if len(s.When.Body.Conditions) != 1 {
		t.Fatalf("expected 1 body condition, got %d", len(s.When.Body.Conditions))
	}
	if s.Policy.RateLimit.Rate != 10 {
		t.Errorf("unexpected rate: %f", s.Policy.RateLimit.Rate)
	}
}
