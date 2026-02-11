package services

import (
	"encoding/json"
	"testing"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

func defaultPaginationConfig() *match.CompiledPagination {
	return &match.CompiledPagination{
		Style:       "page_size",
		PageParam:   "page",
		SizeParam:   "size",
		OffsetParam: "offset",
		LimitParam:  "limit",
		DefaultSize: 3,
		MaxSize:     100,
		DataPath:    "$.items",
		Envelope: match.CompiledPaginationEnvelope{
			DataField:        "data",
			PageField:        "page",
			SizeField:        "size",
			TotalItemsField:  "total_items",
			TotalPagesField:  "total_pages",
			HasNextField:     "has_next",
			HasPreviousField: "has_previous",
		},
	}
}

func TestPaginate_PageSize_FirstPage(t *testing.T) {
	body := []byte(`{"items": [1,2,3,4,5,6,7]}`)
	cfg := defaultPaginationConfig()

	result, err := Paginate(body, cfg, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	assertFloat(t, env, "page", 1)
	assertFloat(t, env, "size", 3)
	assertFloat(t, env, "total_items", 7)
	assertFloat(t, env, "total_pages", 3)
	assertBool(t, env, "has_next", true)
	assertBool(t, env, "has_previous", false)
	assertArrayLen(t, env, "data", 3)
}

func TestPaginate_PageSize_MiddlePage(t *testing.T) {
	body := []byte(`{"items": [1,2,3,4,5,6,7]}`)
	cfg := defaultPaginationConfig()

	result, err := Paginate(body, cfg, map[string]string{"page": "2", "size": "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	assertFloat(t, env, "page", 2)
	assertBool(t, env, "has_next", true)
	assertBool(t, env, "has_previous", true)

	data := env["data"].([]any)
	if len(data) != 3 {
		t.Fatalf("expected 3 items, got %d", len(data))
	}
	// Items should be 4, 5, 6
	if data[0].(float64) != 4 {
		t.Errorf("expected first item to be 4, got %v", data[0])
	}
}

func TestPaginate_PageSize_LastPage(t *testing.T) {
	body := []byte(`{"items": [1,2,3,4,5,6,7]}`)
	cfg := defaultPaginationConfig()

	result, err := Paginate(body, cfg, map[string]string{"page": "3", "size": "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	assertFloat(t, env, "page", 3)
	assertBool(t, env, "has_next", false)
	assertBool(t, env, "has_previous", true)
	assertArrayLen(t, env, "data", 1) // only item 7
}

func TestPaginate_PageSize_BeyondLastPage(t *testing.T) {
	body := []byte(`{"items": [1,2,3,4,5]}`)
	cfg := defaultPaginationConfig()

	result, err := Paginate(body, cfg, map[string]string{"page": "99", "size": "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	assertArrayLen(t, env, "data", 0)
	assertBool(t, env, "has_next", false)
	assertFloat(t, env, "total_items", 5)
}

func TestPaginate_PageSize_MaxSizeCap(t *testing.T) {
	body := []byte(`{"items": [1,2,3,4,5,6,7,8,9,10]}`)
	cfg := defaultPaginationConfig()
	cfg.MaxSize = 4

	result, err := Paginate(body, cfg, map[string]string{"size": "999"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	assertFloat(t, env, "size", 4)
	assertArrayLen(t, env, "data", 4)
}

func TestPaginate_OffsetLimit(t *testing.T) {
	body := []byte(`{"items": [10,20,30,40,50,60,70]}`)
	cfg := defaultPaginationConfig()
	cfg.Style = "offset_limit"

	result, err := Paginate(body, cfg, map[string]string{"offset": "2", "limit": "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	assertFloat(t, env, "size", 3)
	assertFloat(t, env, "total_items", 7)
	assertBool(t, env, "has_next", true)
	assertBool(t, env, "has_previous", true)

	data := env["data"].([]any)
	if data[0].(float64) != 30 {
		t.Errorf("expected first item to be 30, got %v", data[0])
	}
}

func TestPaginate_OffsetLimit_BeyondEnd(t *testing.T) {
	body := []byte(`{"items": [1,2,3]}`)
	cfg := defaultPaginationConfig()
	cfg.Style = "offset_limit"

	result, err := Paginate(body, cfg, map[string]string{"offset": "100", "limit": "5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	assertArrayLen(t, env, "data", 0)
	assertBool(t, env, "has_next", false)
}

func TestPaginate_RootArray(t *testing.T) {
	body := []byte(`[1,2,3,4,5]`)
	cfg := defaultPaginationConfig()
	cfg.DataPath = "$"

	result, err := Paginate(body, cfg, map[string]string{"page": "1", "size": "2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	assertArrayLen(t, env, "data", 2)
	assertFloat(t, env, "total_items", 5)
}

func TestPaginate_EmptyArray(t *testing.T) {
	body := []byte(`{"items": []}`)
	cfg := defaultPaginationConfig()

	result, err := Paginate(body, cfg, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	assertArrayLen(t, env, "data", 0)
	assertFloat(t, env, "total_items", 0)
	assertFloat(t, env, "total_pages", 1)
	assertBool(t, env, "has_next", false)
	assertBool(t, env, "has_previous", false)
}

func TestPaginate_CustomEnvelopeFields(t *testing.T) {
	body := []byte(`{"items": [1,2,3]}`)
	cfg := defaultPaginationConfig()
	cfg.Envelope.DataField = "results"
	cfg.Envelope.TotalItemsField = "count"

	result, err := Paginate(body, cfg, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if _, ok := env["results"]; !ok {
		t.Error("expected 'results' field in envelope")
	}
	if _, ok := env["count"]; !ok {
		t.Error("expected 'count' field in envelope")
	}
	if _, ok := env["data"]; ok {
		t.Error("'data' field should not exist with custom envelope")
	}
}

func TestPaginate_InvalidJSON(t *testing.T) {
	body := []byte(`not json`)
	cfg := defaultPaginationConfig()

	_, err := Paginate(body, cfg, map[string]string{})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestPaginate_NonArrayAtPath(t *testing.T) {
	body := []byte(`{"items": "not an array"}`)
	cfg := defaultPaginationConfig()

	_, err := Paginate(body, cfg, map[string]string{})
	if err == nil {
		t.Fatal("expected error for non-array at data path")
	}
}

func TestPaginate_InvalidQueryParams(t *testing.T) {
	body := []byte(`{"items": [1,2,3,4,5]}`)
	cfg := defaultPaginationConfig()

	// Negative page, non-numeric size — should fall back to defaults
	result, err := Paginate(body, cfg, map[string]string{"page": "-1", "size": "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Defaults: page=1, size=3
	assertFloat(t, env, "page", 1)
	assertFloat(t, env, "size", 3)
}

func TestPaginate_ExactDivision(t *testing.T) {
	body := []byte(`{"items": [1,2,3,4,5,6]}`)
	cfg := defaultPaginationConfig()

	result, err := Paginate(body, cfg, map[string]string{"size": "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env map[string]any
	if err := json.Unmarshal(result, &env); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	assertFloat(t, env, "total_pages", 2)
}

// Helpers

func assertFloat(t *testing.T, m map[string]any, key string, expected float64) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Errorf("missing key %q", key)
		return
	}
	got, ok := v.(float64)
	if !ok {
		t.Errorf("key %q: expected float64, got %T", key, v)
		return
	}
	if got != expected {
		t.Errorf("key %q: expected %v, got %v", key, expected, got)
	}
}

func assertBool(t *testing.T, m map[string]any, key string, expected bool) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Errorf("missing key %q", key)
		return
	}
	got, ok := v.(bool)
	if !ok {
		t.Errorf("key %q: expected bool, got %T", key, v)
		return
	}
	if got != expected {
		t.Errorf("key %q: expected %v, got %v", key, expected, got)
	}
}

func assertArrayLen(t *testing.T, m map[string]any, key string, expected int) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Errorf("missing key %q", key)
		return
	}
	arr, ok := v.([]any)
	if !ok {
		t.Errorf("key %q: expected []any, got %T", key, v)
		return
	}
	if len(arr) != expected {
		t.Errorf("key %q: expected length %d, got %d", key, expected, len(arr))
	}
}
