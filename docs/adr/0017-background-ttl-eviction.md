# ADR-0017: Background TTL Eviction

## Status

Accepted

## Context

Rate limiters are created per key. Over time, one-off keys (from dynamic path parameters or unique request patterns) could accumulate and cause unbounded memory growth.

## Decision

Run background TTL eviction that removes rate limiter entries unused for 10 minutes.

## Consequences

- Prevents unbounded memory growth from one-off keys.
- 10-minute TTL is long enough for active scenarios to retain their state.
- Background goroutine adds minimal overhead.
