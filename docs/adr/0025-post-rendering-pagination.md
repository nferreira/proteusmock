# ADR-0025: Post-Rendering Pagination

## Status

Accepted

## Context

Responses can be both templated and paginated. The system needs to decide whether pagination happens before or after template rendering.

## Decision

Pagination is a post-rendering step. The pipeline is: render body (including templates) -> JSON parse -> slice array -> envelope -> respond.

## Consequences

- Template logic is completely unaware of pagination -- clean separation of concerns.
- Templates can generate the full dataset; pagination handles the slicing.
- Body must be valid JSON after rendering for pagination to work.
