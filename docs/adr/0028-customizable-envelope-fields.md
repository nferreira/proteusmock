# ADR-0028: Customizable Envelope Field Names

## Status

Accepted

## Context

Paginated API responses use envelopes with metadata fields, but different APIs use different field names (e.g., `results` vs `data`, `count` vs `total_items`, `totalPages` vs `total_pages`).

## Decision

All envelope field names are configurable: `data`, `page`, `size`, `total_items`, `total_pages`, `has_next`, `has_previous`. Each has a sensible default.

## Consequences

- Can faithfully mock any API's pagination envelope format.
- Defaults match common conventions, requiring zero configuration for typical cases.
- Increases configuration options, but all are optional.
