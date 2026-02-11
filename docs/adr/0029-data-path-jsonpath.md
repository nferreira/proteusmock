# ADR-0029: data_path JSONPath Extraction

## Status

Accepted

## Context

Not all API responses have a top-level array to paginate. Some nest the data array inside an object (e.g., `{"users": [...]}` or `{"data": {"items": [...]}}`).

## Decision

Use a `data_path` field with JSONPath syntax to select which array to paginate. Default `$` targets the root element, which works for root-level arrays.

## Consequences

- Supports paginating nested arrays without restructuring the response body.
- JSONPath is a well-known query language, familiar to most developers.
- Default `$` requires zero configuration for the common case of root-level arrays.
