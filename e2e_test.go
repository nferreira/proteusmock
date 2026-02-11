package proteusmock_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/domain/trace"
	inboundhttp "github.com/sophialabs/proteusmock/internal/infrastructure/inbound/http"
	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/clock"
	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/filesystem"
	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/ratelimit"
	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/template"
	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
	"github.com/sophialabs/proteusmock/internal/infrastructure/usecases"
	"github.com/sophialabs/proteusmock/internal/testutil"
)

func setupE2EServer(t *testing.T) *httptest.Server {
	t.Helper()

	rootDir := "./mock"
	logger := &testutil.NoopLogger{}
	repo, err := filesystem.NewYAMLRepository(rootDir)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	registry := template.NewRegistry()
	compiler, err := services.NewCompiler(rootDir, registry)
	if err != nil {
		t.Fatalf("failed to create compiler: %v", err)
	}
	clk := clock.New()
	rateLimiterStore := ratelimit.NewTokenBucketStore(10 * time.Minute)
	t.Cleanup(rateLimiterStore.Stop)
	traceBuf := trace.NewRingBuffer(100)
	evaluator := match.NewEvaluator()

	loadUC := usecases.NewLoadScenariosUseCase(repo, compiler, logger)
	handleReqUC := usecases.NewHandleRequestUseCase(evaluator, clk, rateLimiterStore, logger, traceBuf)

	idx, err := loadUC.Execute(context.Background())
	if err != nil {
		t.Fatalf("failed to load scenarios: %v", err)
	}

	server := inboundhttp.NewServer(handleReqUC, loadUC, traceBuf, logger)
	server.Rebuild(idx)

	return httptest.NewServer(server)
}

func TestE2E_HealthCheck(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("GET /api/v1/health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", body["status"])
	}
}

func TestE2E_GetProperties(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	payload := `{"method":{"params":{"contract_id":"100100"}}}`
	resp, err := http.Post(
		ts.URL+"/api/v1/get_properties",
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("POST /api/v1/get_properties failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	props, ok := body["properties"].([]any)
	if !ok {
		t.Fatal("expected properties array")
	}
	if len(props) != 2 {
		t.Errorf("expected 2 properties, got %d", len(props))
	}
}

func TestE2E_GetPropertyByID(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	payload := `{"method":{"params":{"property_id":"1"}}}`
	resp, err := http.Post(
		ts.URL+"/api/v1/get_property",
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("POST /api/v1/get_property failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	prop, ok := body["property"].(map[string]any)
	if !ok {
		t.Fatal("expected property object")
	}
	if prop["name"] != "Main Street Apartments" {
		t.Errorf("unexpected property name: %v", prop["name"])
	}
}

func TestE2E_UnauthorizedFallback(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	// POST without the right body condition should fall through to low-priority unauthorized.
	payload := `{"method":{"params":{"contract_id":"999"}}}`
	resp, err := http.Post(
		ts.URL+"/api/v1/get_properties",
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401 (unauthorized fallback), got %d: %s", resp.StatusCode, body)
	}
}

func TestE2E_NoMatch404Debug(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/nonexistent")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "no_match" {
		t.Errorf("expected 'no_match' error, got %v", body["error"])
	}
}

func TestE2E_ListItems(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/items")
	if err != nil {
		t.Fatalf("GET /api/v1/items failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	items, ok := body["items"].([]any)
	if !ok {
		t.Fatal("expected items array")
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestE2E_AdminListScenarios(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/__admin/scenarios")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var scenarios []map[string]any
	json.NewDecoder(resp.Body).Decode(&scenarios)
	if len(scenarios) < 5 {
		t.Errorf("expected at least 5 scenarios, got %d", len(scenarios))
	}
}

func TestE2E_AdminSearchScenarios(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/__admin/scenarios/search?q=yardi")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var results []map[string]any
	json.NewDecoder(resp.Body).Decode(&results)
	if len(results) < 2 {
		t.Errorf("expected at least 2 yardi scenarios, got %d", len(results))
	}
}

func TestE2E_AdminTrace(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	// Make some requests to populate trace.
	if resp, err := http.Get(ts.URL + "/api/v1/health"); err == nil {
		resp.Body.Close()
	}
	if resp, err := http.Get(ts.URL + "/api/v1/items"); err == nil {
		resp.Body.Close()
	}

	resp, err := http.Get(ts.URL + "/__admin/trace?last=5")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var entries []map[string]any
	json.NewDecoder(resp.Body).Decode(&entries)
	if len(entries) < 2 {
		t.Errorf("expected at least 2 trace entries, got %d", len(entries))
	}
}

func TestE2E_PriorityMatchingOrder(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	// The yardi-get-properties (priority 20) should win over yardi-unauthorized (priority 5)
	// when the body matches.
	payload := `{"method":{"params":{"contract_id":"100100"}}}`
	resp, err := http.Post(
		ts.URL+"/api/v1/get_properties",
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 (high priority match), got %d", resp.StatusCode)
	}
}

func TestE2E_ExprBasics(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/api/v1/users/42?fields=name,email", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer tok_abc")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["id"] != "42" {
		t.Errorf("expected id '42', got %v", body["id"])
	}
	if body["fields"] != "name,email" {
		t.Errorf("expected fields 'name,email', got %v", body["fields"])
	}
	if body["auth"] != "Bearer tok_abc" {
		t.Errorf("expected auth header, got %v", body["auth"])
	}
	if body["request_id"] == nil || body["request_id"] == "" {
		t.Error("expected non-empty request_id (uuid)")
	}
	if body["served_at"] == nil || body["served_at"] == "" {
		t.Error("expected non-empty served_at (timestamp)")
	}
	if body["lucky_number"] == nil {
		t.Error("expected non-nil lucky_number")
	}
}

func TestE2E_ExprConditional(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	tests := []struct {
		name     string
		envValue string
		wantEnv  string
		wantLog  string
	}{
		{"production", "production", "production", "warn"},
		{"staging", "staging", "staging", "debug"},
		{"default", "", "development", "debug"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", ts.URL+"/api/v1/config", nil)
			if tt.envValue != "" {
				req.Header.Set("X-Env", tt.envValue)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			var body map[string]any
			json.NewDecoder(resp.Body).Decode(&body)

			if body["environment"] != tt.wantEnv {
				t.Errorf("expected environment %q, got %v", tt.wantEnv, body["environment"])
			}
			if body["log_level"] != tt.wantLog {
				t.Errorf("expected log_level %q, got %v", tt.wantLog, body["log_level"])
			}
		})
	}
}

func TestE2E_ExprLoops(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/catalog")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)

	items, ok := body["items"].([]any)
	if !ok {
		t.Fatal("expected items array")
	}
	if len(items) != 5 {
		t.Errorf("expected 5 items, got %d", len(items))
	}
}

func TestE2E_ExprEchoBody(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	payload := `{"user": {"name": "Alice", "role": "admin"}}`
	resp, err := http.Post(ts.URL+"/api/v1/echo", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)

	if body["extracted_name"] != "Alice" {
		t.Errorf("expected extracted_name 'Alice', got %v", body["extracted_name"])
	}
	if body["extracted_role"] != "admin" {
		t.Errorf("expected extracted_role 'admin', got %v", body["extracted_role"])
	}
	if body["echo_id"] == nil || body["echo_id"] == "" {
		t.Error("expected non-empty echo_id")
	}
	bodyLen, ok := body["body_length"].(float64)
	if !ok || bodyLen != float64(len(payload)) {
		t.Errorf("expected body_length %d, got %v", len(payload), body["body_length"])
	}
}

func TestE2E_Jinja2Basics(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/submit?source=web", strings.NewReader(""))
	req.Header.Set("X-Request-Id", "req-001")
	req.Header.Set("User-Agent", "TestBot/2.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)

	if body["method"] != "POST" {
		t.Errorf("expected method 'POST', got %v", body["method"])
	}
	if body["source"] != "web" {
		t.Errorf("expected source 'web', got %v", body["source"])
	}
	if body["client_request_id"] != "req-001" {
		t.Errorf("expected client_request_id 'req-001', got %v", body["client_request_id"])
	}
	if body["user_agent"] != "TestBot/2.0" {
		t.Errorf("expected user_agent 'TestBot/2.0', got %v", body["user_agent"])
	}
	if body["server_id"] == nil || body["server_id"] == "" {
		t.Error("expected non-empty server_id (uuid)")
	}
}

func TestE2E_Jinja2Conditional(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	tests := []struct {
		tier       string
		wantTier   string
		wantUpload float64
	}{
		{"premium", "premium", 1000},
		{"basic", "basic", 100},
		{"", "free", 10},
	}

	for _, tt := range tests {
		t.Run(tt.wantTier, func(t *testing.T) {
			req, _ := http.NewRequest("GET", ts.URL+"/api/v1/feature-flags", nil)
			if tt.tier != "" {
				req.Header.Set("X-Tier", tt.tier)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			var body map[string]any
			json.NewDecoder(resp.Body).Decode(&body)

			if body["tier"] != tt.wantTier {
				t.Errorf("expected tier %q, got %v", tt.wantTier, body["tier"])
			}
			if body["max_uploads"] != tt.wantUpload {
				t.Errorf("expected max_uploads %v, got %v", tt.wantUpload, body["max_uploads"])
			}
		})
	}
}

func TestE2E_Jinja2Loops(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/products")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)

	products, ok := body["products"].([]any)
	if !ok {
		t.Fatal("expected products array")
	}
	if len(products) != 4 {
		t.Errorf("expected 4 products, got %d", len(products))
	}

	// Verify first product has expected fields.
	first, ok := products[0].(map[string]any)
	if !ok {
		t.Fatal("expected product object")
	}
	if first["sku"] != "PROD-1" {
		t.Errorf("expected sku 'PROD-1', got %v", first["sku"])
	}
}

func TestE2E_Jinja2EchoBody(t *testing.T) {
	ts := setupE2EServer(t)
	defer ts.Close()

	payload := `{"order": {"id": "ORD-999", "amount": 42.50}}`
	resp, err := http.Post(ts.URL+"/api/v1/process", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "processed" {
		t.Errorf("expected status 'processed', got %v", body["status"])
	}
	if body["order_id"] != "ORD-999" {
		t.Errorf("expected order_id 'ORD-999', got %v", body["order_id"])
	}
	if body["confirmation"] == nil || body["confirmation"] == "" {
		t.Error("expected non-empty confirmation uuid")
	}
	if body["processed_at"] == nil || body["processed_at"] == "" {
		t.Error("expected non-empty processed_at timestamp")
	}
}
