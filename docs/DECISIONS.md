# Design Decisions

No formal ADR files found in repo. Decisions below are derived from code structure, comments, and commit history.

## Architecture

| Decision | Rationale |
|---|---|
| **DDD + Hexagonal Architecture** | Domain layer (`internal/domain/`) has zero external dependencies. All I/O crosses port interfaces. Enables testing without mocks of mocks. |
| **Domain types separate from YAML types** | `scenario.Scenario` (domain) vs `yamlScenario` (infra). Decouples domain model from serialization format. |
| **Compile-once, render-many** | Scenarios compiled to `CompiledScenario` with pre-compiled predicates and templates on load. No YAML parsing or regex compilation on the request path. |
| **Predicate closures** | `type Predicate func(string) bool` -- simple, composable, testable. Boolean combinators (`And`, `Or`, `Not`) compose arbitrarily. |

## Matching

| Decision | Rationale |
|---|---|
| **`METHOD:path-pattern` index key** | O(1) candidate lookup. Only scenarios sharing the same method+path are evaluated per request. |
| **Priority-based ordering** | Higher `priority` matched first. Tie-break: more predicates first (more specific), then alphabetical ID. Deterministic ordering. |
| **First match wins** | Simple, predictable. Combined with priority ordering, gives full control over which scenario matches. |
| **Body predicates receive raw string** | Predicates internally parse JSON/XML. Avoids pre-parsing body in evaluator. Each predicate is self-contained. |

## Template Engines

| Decision | Rationale |
|---|---|
| **Dual engine support (Expr + Jinja2)** | Expr for simple interpolation (`${ }`), Jinja2 for control flow (`{% if %}`, `{% for %}`). Different use cases, user choice. |
| **Explicit `engine` field** | Chosen over auto-detection. Unambiguous, no false positives, zero-cost when not used (static bodies skip template compilation). |
| **`BodyRenderer` interface in domain** | Keeps domain layer clean. Template adapters implement the interface without domain importing template libraries. |
| **Pongo2 for Jinja2** | Mature Go library with Django/Jinja2 syntax. Note: HTML-escapes by default, so raw body embedding via `{{ body }}` produces escaped output for JSON. Use `jsonPath()` instead. |

## Hot Reload

| Decision | Rationale |
|---|---|
| **Atomic pointer swap** | `atomic.Pointer[chi.Mux]` and `atomic.Pointer[ScenarioIndex]`. No mutex on the request path. In-flight requests use old router; new requests use new router. |
| **Debounced file watcher** | `fsnotify` events debounced to 500ms. Prevents rapid recompilation when editors write multiple temp files. |
| **Full reload on any change** | Simpler than incremental. Scenario count is expected to be small (hundreds, not millions). Full reload is fast enough. |

## Rate Limiting

| Decision | Rationale |
|---|---|
| **Token-bucket per key** | `golang.org/x/time/rate.Limiter`. Standard, well-tested. Per-key allows different scenarios to have independent limits. |
| **Background TTL eviction** | Prevents unbounded memory growth from one-off keys. Evicts entries unused for 10 minutes. |
| **Rate/burst update on access** | If a scenario's rate/burst changes (e.g. after hot reload), the limiter is updated in-place rather than recreated. Preserves existing token state. |

## Security

| Decision | Rationale |
|---|---|
| **`filepath.EvalSymlinks` + prefix check** | Prevents path traversal via symlinks in `!include` and `body_file`. Resolved real path must stay within `--root`. |
| **Include depth limit (10)** | Prevents infinite recursion from circular includes. |
| **No authentication** | ProteusMock is a development/test tool, not a production API gateway. Adding auth would add complexity without clear benefit for the target use case. |

## Observability

| Decision | Rationale |
|---|---|
| **Ring buffer for traces** | Fixed memory footprint (default 200 entries). No persistence needed -- traces are ephemeral debugging aids. |
| **Admin API over file-based logs** | `/__admin/trace` and `/__admin/scenarios` endpoints allow live inspection without log parsing. |
| **`slog.Logger` wrapper** | Standard library structured logging. Wrapped behind `ports.Logger` interface for testability. |

## Non-Goals / Trade-offs

| Non-goal | Reasoning |
|---|---|
| **gRPC / WebSocket support** | HTTP-only by design. Keeps scope focused. |
| **Persistent storage** | Scenarios live in YAML files. No database required. |
| **Distributed mode** | Single-process. For multi-instance setups, each instance loads its own YAML files. |
| **Response delay accuracy** | Latency uses `time.Sleep` -- not designed for sub-millisecond precision. Good enough for simulating slow APIs. |
| **Production traffic** | Not a reverse proxy or API gateway. Designed for local dev and CI test environments. |
