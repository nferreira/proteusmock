# ADR-0026: Two Pagination Styles

## Status

Accepted

## Context

Different APIs use different pagination conventions. The two most common patterns are page-based (page number + page size) and offset-based (offset + limit).

## Decision

Support two pagination styles: `page_size` (1-based page number, default) and `offset_limit` (0-based offset). The style is configured via `policy.pagination.style` in the scenario YAML.

## Consequences

- Covers the two most common real-world API pagination patterns.
- Users can mock APIs that use either convention.
- Simple to understand and configure.
