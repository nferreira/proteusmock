# ADR-0006: Priority-Based Ordering

## Status

Accepted

## Context

Multiple scenarios can match the same method and path pattern. A deterministic ordering mechanism is needed to decide which scenario is evaluated first.

## Decision

Higher `priority` value is matched first. Tie-break: scenarios with more predicates are evaluated first (more specific), then alphabetical by ID.

## Consequences

- Users have full control over matching order via the `priority` field.
- Deterministic ordering eliminates ambiguity when multiple scenarios could match.
- Default priority (0) works for simple cases; explicit priority for complex setups.
