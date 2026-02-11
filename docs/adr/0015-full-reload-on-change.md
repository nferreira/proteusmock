# ADR-0015: Full Reload on Any Change

## Status

Accepted

## Context

When a scenario file changes, the system could either incrementally update only the affected scenarios or fully reload everything. Incremental updates would be more efficient but significantly more complex, especially with `!include` dependencies.

## Decision

Perform a full reload on any file change. All scenarios are recompiled from scratch.

## Consequences

- Simple implementation -- no dependency tracking between files.
- Fast enough for expected scenario counts (hundreds, not millions).
- Guarantees consistency -- no stale state from partial updates.
- `!include` changes are automatically picked up without tracking include graphs.
