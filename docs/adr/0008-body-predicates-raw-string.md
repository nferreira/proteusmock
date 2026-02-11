# ADR-0008: Body Predicates Receive Raw String

## Status

Accepted

## Context

Body predicates need to evaluate conditions against JSON and XML request bodies. The system could either pre-parse the body and pass structured data, or pass the raw string and let each predicate parse internally.

## Decision

Body predicates receive the raw body string. Each predicate internally parses JSON/XML as needed.

## Consequences

- Avoids pre-parsing the body in the evaluator -- no wasted parsing if no body predicates exist.
- Each predicate is self-contained and can handle different formats independently.
- Slight overhead from repeated parsing if multiple body predicates exist, but this is negligible for typical use cases.
