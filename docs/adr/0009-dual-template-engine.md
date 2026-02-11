# ADR-0009: Dual Template Engine Support (Expr + Jinja2)

## Status

Accepted

## Context

Response bodies often need dynamic content. Simple interpolation (inserting a header or query param value) requires a lightweight approach, while complex responses (loops, conditionals) need a full template engine.

## Decision

Support two template engines: Expr for simple interpolation (`${ }` syntax) and Jinja2/Pongo2 for control flow (`{% %}` / `{{ }}` syntax). Users choose the engine per response.

## Consequences

- Users pick the right tool for the job -- Expr for simple cases, Jinja2 for complex ones.
- Two template engines to maintain, but they share common template functions.
- Static bodies (no engine specified) skip template compilation entirely -- zero cost.
