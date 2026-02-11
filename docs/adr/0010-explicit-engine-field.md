# ADR-0010: Explicit Engine Field

## Status

Accepted

## Context

The system needs to know which template engine to use for a given response body. Options include auto-detection based on syntax patterns or an explicit field in the scenario definition.

## Decision

Use an explicit `engine` field on the response: `expr`, `jinja2`, or omit for static. A global default can be set via `--default-engine` CLI flag.

## Consequences

- Unambiguous -- no false positives from content that happens to contain `${ }` or `{{ }}`.
- Zero cost for static bodies (no engine field means no template compilation).
- Slightly more verbose scenario definitions when using templates.
