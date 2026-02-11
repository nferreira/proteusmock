# ADR-0022: Ring Buffer for Traces

## Status

Accepted

## Context

Request tracing helps developers debug scenario matching. The trace storage needs a fixed memory footprint since traces are ephemeral debugging aids, not persistent audit logs.

## Decision

Use a ring buffer with a fixed size (default 200 entries) for storing request traces. Oldest entries are overwritten when the buffer is full.

## Consequences

- Fixed, predictable memory usage regardless of request volume.
- No persistence needed -- traces are for live debugging.
- Oldest traces are lost when the buffer wraps, which is acceptable for debugging workflows.
