# Architecture

## Overview

ProteusMock is an HTTP mock server that matches incoming requests against YAML-defined scenarios and returns configured responses. It follows **Domain-Driven Design** with a **Hexagonal (Ports & Adapters)** architecture. The domain layer has zero external dependencies; all I/O crosses port interfaces defined in the infrastructure layer.

Key responsibilities:
- Load and compile YAML scenarios into an in-memory index
- Match HTTP requests against compiled predicates (method, path, headers, body)
- Render static or dynamic (templated) response bodies
- Enforce per-scenario rate limits and latency policies
- Hot-reload scenarios on file changes with zero downtime

## Component Diagram

```mermaid
graph TD
    CLI["cmd/proteusmock<br/><i>CLI entrypoint</i>"]
    APP["internal/app<br/>Config + App<br/><i>lifecycle manager</i>"]
    WIRING["infrastructure/wiring<br/>Container<br/><i>DI composition</i>"]

    CLI --> APP --> WIRING

    WIRING --> HTTP["inbound/http<br/>Server + Router<br/>Admin API<br/>Mock handler"]
    WIRING --> UC["usecases/<br/>LoadScenarios<br/>HandleRequest"]
    WIRING --> OUT["outbound/"]

    UC --> SVC["services/<br/>Compiler<br/>ScenarioIndex<br/>ContentType"]

    OUT --- FS["filesystem/<br/>YAMLRepo · Watcher<br/>IncludeResolver"]
    OUT --- TPL["template/<br/>ExprCompiler<br/>Jinja2Compiler · Registry"]
    OUT --- ADAPTERS["clock/ · logging/<br/>ratelimit/"]

    SVC --> SCENARIO["domain/scenario<br/>Scenario · Repository<br/>WhenClause · BodyClause<br/>Policy"]
    SVC --> MATCH["domain/match<br/>Evaluator · Predicate<br/>CompiledScenario<br/>BodyRenderer"]
    SVC --> TRACE["domain/trace<br/>RingBuffer · Entry"]

    style CLI fill:#4a9eff,color:#fff
    style APP fill:#4a9eff,color:#fff
    style WIRING fill:#6c7ae0,color:#fff
    style HTTP fill:#2ecc71,color:#fff
    style UC fill:#2ecc71,color:#fff
    style OUT fill:#2ecc71,color:#fff
    style SVC fill:#e67e22,color:#fff
    style SCENARIO fill:#e74c3c,color:#fff
    style MATCH fill:#e74c3c,color:#fff
    style TRACE fill:#e74c3c,color:#fff
```

## Package Map

| Package | Role | Key Types | Notes |
|---|---|---|---|
| `cmd/proteusmock` | CLI entrypoint | `main()` | Parses flags, creates `App`, calls `Run` |
| `internal/app` | Lifecycle | `Config`, `App` | Owns startup, shutdown, watcher setup |
| `internal/domain/scenario` | Domain entities | `Scenario`, `Repository`, `WhenClause`, `BodyClause`, `Policy` | Zero deps |
| `internal/domain/match` | Matching engine | `Evaluator`, `Predicate`, `CompiledScenario`, `BodyRenderer`, `RenderContext` | Zero deps |
| `internal/domain/trace` | Request tracing | `RingBuffer`, `Entry`, `CandidateResult` | Thread-safe ring buffer |
| `internal/infrastructure/ports` | Port interfaces | `Clock`, `Logger`, `RateLimiter` | Contracts for adapters |
| `internal/infrastructure/services` | Compilation & indexing | `Compiler`, `ScenarioIndex`, `Paginate`, `InferContentType` | Compiles YAML into predicates, paginates responses |
| `internal/infrastructure/usecases` | Application logic | `LoadScenariosUseCase`, `HandleRequestUseCase` | Orchestrates domain + infra |
| `internal/infrastructure/inbound/http` | HTTP adapter | `Server` | chi router, admin & mock handlers |
| `internal/infrastructure/outbound/filesystem` | YAML adapter | `YAMLRepository`, `Watcher`, `IncludeResolver` | Implements `scenario.Repository` |
| `internal/infrastructure/outbound/template` | Template engines | `Registry`, `ExprCompiler`, `Jinja2Compiler` | Implements `match.BodyRenderer` |
| `internal/infrastructure/outbound/clock` | Clock adapter | `RealClock` | Implements `ports.Clock` |
| `internal/infrastructure/outbound/logging` | Log adapter | `SlogLogger` | Wraps `slog.Logger` |
| `internal/infrastructure/outbound/ratelimit` | Rate-limit adapter | `TokenBucketStore` | Per-key token bucket with TTL eviction |
| `internal/infrastructure/wiring` | DI container | `Container`, `Params` | Constructs and wires all components |
| `internal/testutil` | Test fakes | `NoopLogger`, `FixedClock`, `StubRateLimiter`, `StubBodyRenderer` | Shared across test packages |

## Request Flow

```mermaid
flowchart TD
    REQ([HTTP Request])
    SERVE["Server.ServeHTTP<br/>Load router via atomic.Pointer"]
    ROUTE{"chi.Mux<br/>route dispatch"}
    ADMIN["Admin handler"]
    MOCK["mockHandler"]

    REQ --> SERVE --> ROUTE
    ROUTE -->|/__admin/*| ADMIN
    ROUTE -->|mock route| MOCK

    subgraph mockHandler
        READ["1. Read body ≤ 10 MB"]
        BUILD["2. Build IncomingRequest<br/>{Method, Path, Headers, Body}"]
        PATTERN["3. Resolve route pattern<br/>via chi.RouteContext"]
        LOOKUP["4. Lookup candidates<br/>index·METHOD:path-pattern·"]
        EXEC["5. Execute HandleRequestUseCase"]
        READ --> BUILD --> PATTERN --> LOOKUP --> EXEC
    end

    subgraph HandleRequestUseCase.Execute
        EVAL["Evaluator.Evaluate<br/>candidates sorted by priority<br/>first full match wins"]
        TRACE_ENTRY["Record trace entry"]
        MATCH_CHECK{Match<br/>found?}
        NO_MATCH["Return Matched: false"]
        RATE["Check rate limit"]
        LATENCY["Apply latency<br/>context-aware"]
        CONTENT["Infer content type"]
        RESULT["Return response<br/>+ pagination config"]

        EVAL --> TRACE_ENTRY --> MATCH_CHECK
        MATCH_CHECK -->|No| NO_MATCH
        MATCH_CHECK -->|Yes| RATE --> LATENCY --> CONTENT --> RESULT
    end

    EXEC --> EVAL

    subgraph Response Pipeline
        RENDER_CHECK{Renderer<br/>present?}
        RENDER["Render dynamic body"]
        PAGE_CHECK{Pagination<br/>present?}
        PAGE["Paginate: parse JSON →<br/>extract array at data_path →<br/>slice by query params →<br/>wrap in envelope"]
        WRITE["Write status + headers + body"]

        RENDER_CHECK -->|Yes| RENDER --> PAGE_CHECK
        RENDER_CHECK -->|No| PAGE_CHECK
        PAGE_CHECK -->|Yes| PAGE --> WRITE
        PAGE_CHECK -->|No| WRITE
    end

    RESULT --> RENDER_CHECK
    NO_MATCH --> WRITE_404["Write 404 response"]
```

## Scenario Load Flow

```mermaid
flowchart TD
    TRIGGER(["App.Run / Watcher trigger"])

    subgraph LoadScenariosUseCase.Execute
        LOAD["repo.LoadAll()"]
        WALK["Walk YAML files"]
        PARSE["Parse yaml.Node tree"]
        INCLUDE["Resolve !include tags<br/>recursive, max depth 10"]
        DECODE["Decode → scenario.Scenario"]
        ENGINE["Apply default engine<br/>if configured"]
        VALIDATE["Validate unique IDs"]
        COMPILE["For each scenario:<br/>compiler.CompileScenario()"]
        WHEN_C["Compile when clause<br/>→ FieldPredicates"]
        BODY_C["Compile body clause<br/>→ nested Predicate closures"]
        RESP_C["Resolve response body<br/>inline / file / template"]
        TPL_C["Compile template<br/>via Registry if engine set"]
        INDEX_ADD["Add to ScenarioIndex"]
        BUILD["index.Build()<br/>sort by priority desc"]

        LOAD --> WALK --> PARSE --> INCLUDE --> DECODE
        DECODE --> ENGINE --> VALIDATE --> COMPILE
        COMPILE --> WHEN_C & BODY_C & RESP_C
        RESP_C --> TPL_C
        WHEN_C & BODY_C & TPL_C --> INDEX_ADD --> BUILD
    end

    subgraph Server.Rebuild
        NEW_MUX["Build new chi.Mux<br/>with all routes"]
        SWAP["Atomic swap<br/>router + index pointers"]
        NEW_MUX --> SWAP
    end

    TRIGGER --> LOAD
    BUILD --> NEW_MUX
```

## Key Abstractions (Ports & Adapters)

| Port (Interface) | Location | Adapter | Location |
|---|---|---|---|
| `scenario.Repository` | `domain/scenario/` | `YAMLRepository` | `outbound/filesystem/` |
| `match.BodyRenderer` | `domain/match/` | `exprRenderer`, `jinja2Renderer` | `outbound/template/` |
| `ports.Clock` | `infrastructure/ports/` | `RealClock` | `outbound/clock/` |
| `ports.Logger` | `infrastructure/ports/` | `SlogLogger` | `outbound/logging/` |
| `ports.RateLimiter` | `infrastructure/ports/` | `TokenBucketStore` | `outbound/ratelimit/` |
| `services.TemplateRegistry` | `infrastructure/services/` | `Registry` | `outbound/template/` |

## Hot Reload

1. `fsnotify.Watcher` monitors root directory tree
2. Events debounced (default 500ms) to avoid rapid recompilation
3. Reload callback: `LoadScenariosUseCase.Execute()` -> `Server.Rebuild()`
4. `atomic.Pointer[chi.Mux]` and `atomic.Pointer[ScenarioIndex]` ensure zero-downtime swap
5. New directories are auto-watched

## Concurrency Model

- **Main goroutine**: HTTP server via `http.Server.ListenAndServe`
- **Watcher goroutine**: file-change listener + debounce timer
- **Rate-limiter eviction goroutine**: periodic cleanup of stale entries
- **Atomic pointers**: router and index swapped atomically (no mutex)
- **RingBuffer**: mutex-protected for concurrent trace writes/reads

## Security

- **Path traversal prevention**: `!include` and `body_file` paths validated via `filepath.EvalSymlinks` + root prefix check
- **Include depth limit**: max 10 levels of nesting
- **Body size limit**: 10 MB max request body
- **No authentication** on mock or admin endpoints (by design -- intended for local/test use)

## Observability

| Signal | Implementation | Access |
|---|---|---|
| Structured logging | `slog.Logger` via `ports.Logger` | stderr (configurable level) |
| Request tracing | `trace.RingBuffer` (fixed size, default 200) | `GET /__admin/trace?last=N` |
| Scenario inspection | Admin API | `GET /__admin/scenarios` |
| Scenario search | Admin API | `GET /__admin/scenarios/search?q=term` |
