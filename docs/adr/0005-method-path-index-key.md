# ADR-0005: METHOD:path-pattern Index Key

## Status

Accepted

## Context

When an HTTP request arrives, the system must quickly find candidate scenarios to evaluate. Scanning all scenarios linearly would be inefficient.

## Decision

Index scenarios by `METHOD:path-pattern` key in a `ScenarioIndex` map. Only scenarios sharing the same method and path pattern are evaluated per request, achieving O(1) candidate lookup.

## Consequences

- Fast candidate lookup regardless of total scenario count.
- Path patterns must be normalized consistently to serve as map keys.
- Chi route pattern lookup (`rctx.RoutePattern()`) is needed for index key matching with path parameters.
