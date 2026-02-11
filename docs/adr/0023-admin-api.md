# ADR-0023: Admin API over File-Based Logs

## Status

Accepted

## Context

Developers need to inspect loaded scenarios and recent request traces. This could be done via log files or a live API.

## Decision

Provide `/__admin/trace` and `/__admin/scenarios` HTTP endpoints for live inspection rather than relying on file-based log parsing.

## Consequences

- Live inspection without log file access or parsing tools.
- Integrates naturally with HTTP-based workflows (curl, browser, scripts).
- Admin endpoints share the same HTTP server -- no additional ports or processes.
