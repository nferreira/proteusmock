# ADR-0027: Customizable Query Param Names

## Status

Accepted

## Context

Real APIs use different query parameter names for pagination. Some use `?page=&size=`, others use `?page_number=&page_size=`, `?offset=&count=`, etc.

## Decision

Allow customizing query parameter names via `page_param`, `size_param`, `offset_param`, and `limit_param` configuration fields. Defaults (`page`, `size`, `offset`, `limit`) work out of the box.

## Consequences

- Can mock any API's pagination parameter convention.
- Sensible defaults require zero configuration for common cases.
- Slightly more configuration surface, but optional.
