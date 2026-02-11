# ADR-0007: First Match Wins

## Status

Accepted

## Context

After ordering candidates by priority, the system needs a strategy for selecting which scenario to use when multiple candidates could match.

## Decision

Use a first-match-wins strategy. The first scenario in priority order whose predicates all pass is selected.

## Consequences

- Simple and predictable matching behavior.
- Combined with priority-based ordering (ADR-0006), users have full control over which scenario matches.
- No complex scoring or weighted matching needed.
