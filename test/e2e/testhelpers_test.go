//go:build e2e

package e2e_test

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"runtime"
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

func projectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	// file = <root>/test/e2e/testhelpers_test.go → go up 3 levels
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func setupE2EServer(t *testing.T) *httptest.Server {
	t.Helper()

	rootDir := filepath.Join(projectRoot(), "mock")
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
