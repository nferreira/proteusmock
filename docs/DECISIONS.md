# Design Decisions

Formal ADR files are available in [`docs/adr/`](adr/README.md). The summary below is derived from those records.

## Architecture

| Decision | ADR | Rationale |
|---|---|---|
| **DDD + Hexagonal Architecture** | [ADR-0001](adr/0001-ddd-hexagonal-architecture.md) | Domain layer (`internal/domain/`) has zero external dependencies. All I/O crosses port interfaces. Enables testing without mocks of mocks. |
| **Domain types separate from YAML types** | [ADR-0002](adr/0002-domain-types-separate-from-yaml.md) | `scenario.Scenario` (domain) vs `yamlScenario` (infra). Decouples domain model from serialization format. |
| **Compile-once, render-many** | [ADR-0003](adr/0003-compile-once-render-many.md) | Scenarios compiled to `CompiledScenario` with pre-compiled predicates and templates on load. No YAML parsing or regex compilation on the request path. |
| **Predicate closures** | [ADR-0004](adr/0004-predicate-closures.md) | `type Predicate func(string) bool` -- simple, composable, testable. Boolean combinators (`And`, `Or`, `Not`) compose arbitrarily. |

## Matching

| Decision | ADR | Rationale |
|---|---|---|
| **`METHOD:path-pattern` index key** | [ADR-0005](adr/0005-method-path-index-key.md) | O(1) candidate lookup. Only scenarios sharing the same method+path are evaluated per request. |
| **Priority-based ordering** | [ADR-0006](adr/0006-priority-based-ordering.md) | Higher `priority` matched first. Tie-break: more predicates first (more specific), then alphabetical ID. Deterministic ordering. |
| **First match wins** | [ADR-0007](adr/0007-first-match-wins.md) | Simple, predictable. Combined with priority ordering, gives full control over which scenario matches. |
| **Body predicates receive raw string** | [ADR-0008](adr/0008-body-predicates-raw-string.md) | Predicates internally parse JSON/XML. Avoids pre-parsing body in evaluator. Each predicate is self-contained. |

## Template Engines

| Decision | ADR | Rationale |
|---|---|---|
| **Dual engine support (Expr + Jinja2)** | [ADR-0009](adr/0009-dual-template-engine.md) | Expr for simple interpolation (`${ }`), Jinja2 for control flow (`{% if %}`, `{% for %}`). Different use cases, user choice. |
| **Explicit `engine` field** | [ADR-0010](adr/0010-explicit-engine-field.md) | Chosen over auto-detection. Unambiguous, no false positives, zero-cost when not used (static bodies skip template compilation). |
| **`BodyRenderer` interface in domain** | [ADR-0011](adr/0011-body-renderer-interface.md) | Keeps domain layer clean. Template adapters implement the interface without domain importing template libraries. |
| **Pongo2 for Jinja2** | [ADR-0012](adr/0012-pongo2-for-jinja2.md) | Mature Go library with Django/Jinja2 syntax. Note: HTML-escapes by default, so raw body embedding via `{{ body }}` produces escaped output for JSON. Use `jsonPath()` instead. |

## Hot Reload

| Decision | ADR | Rationale |
|---|---|---|
| **Atomic pointer swap** | [ADR-0013](adr/0013-atomic-pointer-swap.md) | `atomic.Pointer[chi.Mux]` and `atomic.Pointer[ScenarioIndex]`. No mutex on the request path. In-flight requests use old router; new requests use new router. |
| **Debounced file watcher** | [ADR-0014](adr/0014-debounced-file-watcher.md) | `fsnotify` events debounced to 500ms. Prevents rapid recompilation when editors write multiple temp files. |
| **Full reload on any change** | [ADR-0015](adr/0015-full-reload-on-change.md) | Simpler than incremental. Scenario count is expected to be small (hundreds, not millions). Full reload is fast enough. |

## Rate Limiting

| Decision | ADR | Rationale |
|---|---|---|
| **Token-bucket per key** | [ADR-0016](adr/0016-token-bucket-rate-limiting.md) | `golang.org/x/time/rate.Limiter`. Standard, well-tested. Per-key allows different scenarios to have independent limits. |
| **Background TTL eviction** | [ADR-0017](adr/0017-background-ttl-eviction.md) | Prevents unbounded memory growth from one-off keys. Evicts entries unused for 10 minutes. |
| **Rate/burst update on access** | [ADR-0018](adr/0018-rate-burst-update-on-access.md) | If a scenario's rate/burst changes (e.g. after hot reload), the limiter is updated in-place rather than recreated. Preserves existing token state. |

## Security

| Decision | ADR | Rationale |
|---|---|---|
| **`filepath.EvalSymlinks` + prefix check** | [ADR-0019](adr/0019-symlink-path-traversal-prevention.md) | Prevents path traversal via symlinks in `!include` and `body_file`. Resolved real path must stay within `--root`. |
| **Include depth limit (10)** | [ADR-0020](adr/0020-include-depth-limit.md) | Prevents infinite recursion from circular includes. |
| **No authentication** | [ADR-0021](adr/0021-no-authentication.md) | ProteusMock is a development/test tool, not a production API gateway. Adding auth would add complexity without clear benefit for the target use case. |

## Observability

| Decision | ADR | Rationale |
|---|---|---|
| **Ring buffer for traces** | [ADR-0022](adr/0022-ring-buffer-traces.md) | Fixed memory footprint (default 200 entries). No persistence needed -- traces are ephemeral debugging aids. |
| **Admin API over file-based logs** | [ADR-0023](adr/0023-admin-api.md) | `/__admin/trace` and `/__admin/scenarios` endpoints allow live inspection without log parsing. |
| **`slog.Logger` wrapper** | [ADR-0024](adr/0024-slog-logger-wrapper.md) | Standard library structured logging. Wrapped behind `ports.Logger` interface for testability. |

## Pagination

| Decision | ADR | Rationale |
|---|---|---|
| **Post-rendering pagination** | [ADR-0025](adr/0025-post-rendering-pagination.md) | Body is rendered first (including templates), then parsed, sliced, and enveloped. Keeps template logic unaware of pagination. Simple pipeline: render -> paginate -> respond. |
| **Two styles: page_size and offset_limit** | [ADR-0026](adr/0026-two-pagination-styles.md) | Covers the two most common API pagination patterns. `page_size` (1-based page number) is the default. `offset_limit` (0-based offset) for APIs that prefer cursor-less offset pagination. |
| **Customizable query param names** | [ADR-0027](adr/0027-customizable-query-param-names.md) | `page_param`, `size_param`, `offset_param`, `limit_param` allow matching any API convention (e.g. `?page_size=` instead of `?size=`). Defaults (`page`, `size`, `offset`, `limit`) work out of the box. |
| **Customizable envelope field names** | [ADR-0028](adr/0028-customizable-envelope-fields.md) | Real APIs use different field names (`results` vs `data`, `count` vs `total_items`). All envelope fields are configurable with sensible defaults. |
| **`data_path` JSONPath extraction** | [ADR-0029](adr/0029-data-path-jsonpath.md) | Allows paginating a nested array (e.g. `$.users`) rather than requiring the root to be an array. Default `$` works for root-level arrays. |
| **Graceful degradation** | [ADR-0030](adr/0030-pagination-graceful-degradation.md) | Pagination errors (invalid JSON, missing array, bad data_path) log a warning and return the original unpaginated body. Avoids breaking responses when pagination config doesn't match the body structure. |
| **Pure function design** | [ADR-0031](adr/0031-pagination-pure-function.md) | `services.Paginate(body, config, queryParams) -> body` is a pure function with no side effects. Easy to test, no state. |

## Non-Goals / Trade-offs

| Non-goal | Reasoning |
|---|---|
| **gRPC / WebSocket support** | HTTP-only by design. Keeps scope focused. |
| **Persistent storage** | Scenarios live in YAML files. No database required. |
| **Distributed mode** | Single-process. For multi-instance setups, each instance loads its own YAML files. |
| **Response delay accuracy** | Latency uses `time.Sleep` -- not designed for sub-millisecond precision. Good enough for simulating slow APIs. |
| **Production traffic** | Not a reverse proxy or API gateway. Designed for local dev and CI test environments. |
