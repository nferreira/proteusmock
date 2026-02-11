# ADR-0003: Compile-Once, Render-Many

## Status

Accepted

## Context

Scenarios are loaded from YAML files and matched against incoming HTTP requests. Repeatedly parsing YAML, compiling regexes, and building predicates on every request would add unnecessary latency.

## Decision

Compile scenarios to `CompiledScenario` structs with pre-compiled predicates and templates at load time. No YAML parsing or regex compilation occurs on the request path.

## Consequences

- Request handling is fast -- only evaluation of pre-compiled predicates.
- Slightly higher memory usage for compiled structures.
- Hot reload must recompile all scenarios, but this is acceptable given the expected scenario count (hundreds, not millions).
