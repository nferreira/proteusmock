# Extending ProteusMock

## Adding a Feature End-to-End

The codebase follows Hexagonal Architecture. The typical flow for a new feature:

```
1. Define interface (port)     -> domain/ or infrastructure/ports/
2. Implement adapter           -> infrastructure/outbound/
3. Wire in DI container        -> infrastructure/wiring/container.go
4. Add use-case logic          -> infrastructure/usecases/
5. Expose via HTTP (if needed) -> infrastructure/inbound/http/server.go
6. Write tests                 -> *_test.go alongside implementation
```

## Adding a New Template Engine

**Example**: adding a `handlebars` engine.

### 1. Create adapter

```go
// internal/infrastructure/outbound/template/handlebars.go
package template

import "github.com/sophialabs/proteusmock/internal/domain/match"

type HandlebarsCompiler struct{}

func (h *HandlebarsCompiler) Compile(name, source string) (match.BodyRenderer, error) {
    // Parse template, return a BodyRenderer that implements:
    //   Render(ctx match.RenderContext) ([]byte, error)
}
```

The `BodyRenderer` interface (in `domain/match/predicate.go`):

```go
type BodyRenderer interface {
    Render(ctx RenderContext) ([]byte, error)
}
```

### 2. Register in Registry

```go
// internal/infrastructure/outbound/template/registry.go
func NewRegistry() *Registry {
    return &Registry{
        engines: map[string]EngineCompiler{
            "expr":       &ExprCompiler{},
            "jinja2":     &Jinja2Compiler{},
            "handlebars": &HandlebarsCompiler{},  // add here
        },
    }
}
```

No wiring changes needed -- the `Registry` is already injected into the `Compiler`.

### 3. Use in scenarios

```yaml
response:
  engine: handlebars
  body: |
    {"greeting": "Hello {{name}}!"}
```

## Adding a New Admin Endpoint

Admin routes live in `internal/infrastructure/inbound/http/server.go` inside `BuildRouter()`:

```go
func (s *Server) BuildRouter(idx *services.ScenarioIndex) *chi.Mux {
    r := chi.NewRouter()

    // Admin routes
    r.Route("/__admin", func(r chi.Router) {
        r.Get("/scenarios", s.listScenarios)
        r.Get("/scenarios/search", s.searchScenarios)
        r.Get("/trace", s.getTrace)
        r.Post("/reload", s.reloadScenarios)
        // Add new endpoint here:
        r.Get("/stats", s.getStats)
    })
    // ...
}
```

Add the handler method on `*Server`:

```go
func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
    // Access s.traceBuf, s.idx.Load(), etc.
}
```

## Adding a New Scenario Repository

The repository port is defined in `domain/scenario/repository.go`:

```go
type Repository interface {
    LoadAll(ctx context.Context) ([]*Scenario, error)
}
```

### 1. Implement the interface

```go
// internal/infrastructure/outbound/database/repository.go
type PostgresRepository struct { db *sql.DB }

func (r *PostgresRepository) LoadAll(ctx context.Context) ([]*scenario.Scenario, error) {
    // Query database, map rows to scenario.Scenario
}
```

### 2. Wire in container

Update `internal/infrastructure/wiring/container.go`:

```go
func New(p Params) (*Container, error) {
    // Replace or compose with existing repository:
    repo, err := database.NewPostgresRepository(p.DSN)
    // Pass repo to LoadScenariosUseCase
}
```

## Adding a New Port (Interface)

Ports live in `internal/infrastructure/ports/ports.go`:

```go
type Clock interface {
    Now() time.Time
    SleepContext(ctx context.Context, d time.Duration) error
}

type Logger interface {
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    Debug(msg string, args ...any)
}

type RateLimiter interface {
    Allow(ctx context.Context, key string, rate float64, burst int) bool
}
```

To add a new port:
1. Define interface in `ports/ports.go`
2. Create adapter in `outbound/<name>/`
3. Create test fake in `internal/testutil/fakes.go`
4. Accept in `wiring.Params` or construct in `wiring.New()`

## Patterns Used

| Pattern | Where | Purpose |
|---|---|---|
| **Ports & Adapters** | `ports/` interfaces + `outbound/` implementations | Testability, swappable deps |
| **Repository** | `scenario.Repository` -> `YAMLRepository` | Abstract storage |
| **Compile-once, render-many** | `Compiler` -> `CompiledScenario` | Avoid re-parsing per request |
| **Predicate closures** | `match.Predicate = func(string) bool` | Composable matching logic |
| **Boolean combinators** | `And()`, `Or()`, `Not()` on predicates | Recursive body matching |
| **Atomic pointer swap** | `atomic.Pointer[chi.Mux]` in `Server` | Zero-downtime hot reload |
| **Ring buffer** | `trace.RingBuffer` | Bounded trace storage |
| **Token bucket** | `ratelimit.TokenBucketStore` | Per-key rate limiting |
| **DI container** | `wiring.Container` | Controlled construction order |
| **Use cases** | `usecases/` package | Orchestrate domain + infra logic |

## Testing Strategy

### Test types

| Type | Location | Run with |
|---|---|---|
| Unit tests | `*_test.go` alongside source | `go test ./internal/...` |
| E2E tests | `e2e_test.go` (root) | `go test -run TestE2E ./...` |
| All tests | | `make test` or `make test-race` |

### Test fakes (`internal/testutil/fakes.go`)

| Fake | Implements | Behavior |
|---|---|---|
| `NoopLogger` | `ports.Logger` | Discards all output |
| `FixedClock` | `ports.Clock` | Returns fixed time, no sleep |
| `StubRateLimiter` | `ports.RateLimiter` | `AllowAll: true/false` |
| `StubBodyRenderer` | `match.BodyRenderer` | Returns configured `Result`/`Err` |

### Writing a new test

```go
func TestMyFeature(t *testing.T) {
    logger := &testutil.NoopLogger{}
    clock  := &testutil.FixedClock{T: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
    rl     := &testutil.StubRateLimiter{AllowAll: true}

    // Build components with fakes
    // Assert behavior
}
```

### E2E tests

E2E tests use `httptest.NewServer` with the full wiring:

```go
func TestE2E_Something(t *testing.T) {
    // Setup: create temp dir, write YAML files
    // Build container with test config
    // Start httptest server
    // Make HTTP requests and assert responses
}
```

### Running tests

```bash
make test           # go test ./...
make test-race      # go test -race -count=1 ./...
make test-cover     # coverage report
```
