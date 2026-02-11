# ADR-0024: slog.Logger Wrapper

## Status

Accepted

## Context

The project needs structured logging that is testable and does not couple the domain layer to a specific logging library.

## Decision

Use the standard library `slog.Logger` for structured logging, wrapped behind a `ports.Logger` interface for testability.

## Consequences

- No external logging dependency -- uses Go standard library.
- `ports.Logger` interface allows injecting a no-op logger in tests.
- Structured logging with key-value pairs for machine-parseable output.
