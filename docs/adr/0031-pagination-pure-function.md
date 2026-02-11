# ADR-0031: Pagination Pure Function Design

## Status

Accepted

## Context

The pagination logic needs to transform a rendered response body into a paginated response. The design should be testable and free of side effects.

## Decision

Implement pagination as a pure function: `services.Paginate(body []byte, config CompiledPagination, queryParams url.Values) []byte`. No side effects, no state.

## Consequences

- Trivially testable -- provide inputs, assert outputs.
- No hidden dependencies or state to manage.
- Can be called from any context without setup or teardown.
- Composable within the render pipeline.
