# ADR-0016: Token-Bucket Rate Limiting

## Status

Accepted

## Context

Scenarios can define rate limits to simulate throttled APIs. The rate limiting algorithm needs to be well-understood, standard, and support per-key (per-scenario) independent limits.

## Decision

Use token-bucket rate limiting via `golang.org/x/time/rate.Limiter`. Each scenario key gets its own independent limiter.

## Consequences

- Standard, well-tested algorithm from the Go extended standard library.
- Per-key limiters allow different scenarios to have independent rate limits.
- Supports burst capacity naturally via the token bucket model.
