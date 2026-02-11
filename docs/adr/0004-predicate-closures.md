# ADR-0004: Predicate Closures

## Status

Accepted

## Context

Scenario matching requires evaluating conditions on headers, query parameters, path parameters, and request body. The predicate system needs to be simple, composable, and testable.

## Decision

Use `type Predicate func(string) bool` -- simple closures. Boolean combinators (`And`, `Or`, `Not`) compose predicates arbitrarily.

## Consequences

- Predicates are trivially testable -- just call them with a string.
- Arbitrary composition via combinators without complex type hierarchies.
- No need for a predicate DSL or expression evaluator for conditions.
- Each predicate is self-contained and stateless.
