# ADR-0013: Atomic Pointer Swap for Hot Reload

## Status

Accepted

## Context

When scenarios are reloaded (e.g., after file changes), the router and scenario index must be updated. This must be safe for concurrent access -- in-flight requests should not see partially updated state.

## Decision

Use `atomic.Pointer[chi.Mux]` and `atomic.Pointer[ScenarioIndex]` for lock-free updates. In-flight requests use the old router; new requests use the new router after the swap.

## Consequences

- No mutex on the request path -- zero contention for reads.
- Atomic swap guarantees consistency: a request either sees the old state or the new state, never a mix.
- Slightly more complex than a simple mutex, but much better performance under load.
