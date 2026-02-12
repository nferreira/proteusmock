package wiring

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/domain/trace"
	inboundhttp "github.com/sophialabs/proteusmock/internal/infrastructure/inbound/http"
	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/clock"
	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/filesystem"
	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/ratelimit"
	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/template"
	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
	"github.com/sophialabs/proteusmock/internal/infrastructure/usecases"
)

// Params holds the subset of configuration needed to construct infrastructure components.
type Params struct {
	RootDir        string
	TraceSize      int
	RateLimiterTTL time.Duration
	Logger         ports.Logger
	DefaultEngine  string // "" = static, "expr", "jinja2"
}

// Container owns the construction and lifecycle of all infrastructure components.
type Container struct {
	logger           ports.Logger
	server           *inboundhttp.Server
	loadUC           *usecases.LoadScenariosUseCase
	saveUC           *usecases.SaveScenarioUseCase
	deleteUC         *usecases.DeleteScenarioUseCase
	rateLimiterStore *ratelimit.TokenBucketStore
	traceBuf         *trace.RingBuffer
	closeOnce        sync.Once
}

// New constructs all infrastructure components. Fallible operations (repository,
// compiler) run before goroutine-starting operations (rate limiter store) to
// avoid goroutine leaks on early failure.
func New(p Params) (*Container, error) {
	if _, err := os.Stat(p.RootDir); err != nil {
		return nil, fmt.Errorf("failed to access root directory: %w", err)
	}

	repo, err := filesystem.NewYAMLRepository(p.RootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	registry := template.NewRegistry()
	compiler, err := services.NewCompiler(p.RootDir, registry)
	if err != nil {
		return nil, fmt.Errorf("failed to create compiler: %w", err)
	}

	// Start background goroutine only after all fallible ops succeed.
	rateLimiterStore := ratelimit.NewTokenBucketStore(p.RateLimiterTTL)

	clk := clock.New()
	traceBuf := trace.NewRingBuffer(p.TraceSize)
	evaluator := match.NewEvaluator()

	loadUC := usecases.NewLoadScenariosUseCase(repo, compiler, p.Logger)
	if p.DefaultEngine != "" {
		loadUC.SetDefaultEngine(p.DefaultEngine)
	}
	handleReqUC := usecases.NewHandleRequestUseCase(evaluator, clk, rateLimiterStore, p.Logger, traceBuf)
	saveUC := usecases.NewSaveScenarioUseCase(repo, p.Logger)
	deleteUC := usecases.NewDeleteScenarioUseCase(repo, p.Logger)

	server := inboundhttp.NewServer(handleReqUC, loadUC, traceBuf, p.Logger)
	server.SetCRUDDeps(saveUC, deleteUC, repo, p.RootDir)

	return &Container{
		logger:           p.Logger,
		server:           server,
		loadUC:           loadUC,
		saveUC:           saveUC,
		deleteUC:         deleteUC,
		rateLimiterStore: rateLimiterStore,
		traceBuf:         traceBuf,
	}, nil
}

// Close releases resources held by the container. It is idempotent.
func (c *Container) Close() {
	c.closeOnce.Do(func() {
		c.rateLimiterStore.Stop()
	})
}

// Logger returns the logger passed at construction time.
func (c *Container) Logger() ports.Logger {
	return c.logger
}

// Server returns the HTTP mock server.
func (c *Container) Server() *inboundhttp.Server {
	return c.server
}

// LoadScenariosUseCase returns the use case for loading and compiling scenarios.
func (c *Container) LoadScenariosUseCase() *usecases.LoadScenariosUseCase {
	return c.loadUC
}

// RateLimiterStore returns the token bucket store for rate limiting.
func (c *Container) RateLimiterStore() *ratelimit.TokenBucketStore {
	return c.rateLimiterStore
}

// TraceBuf returns the trace ring buffer.
func (c *Container) TraceBuf() *trace.RingBuffer {
	return c.traceBuf
}
