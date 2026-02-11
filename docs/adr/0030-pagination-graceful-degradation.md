# ADR-0030: Pagination Graceful Degradation

## Status

Accepted

## Context

Pagination requires the response body to be valid JSON containing an array at the specified `data_path`. If any of these assumptions fail (invalid JSON, missing array, bad data_path), the system must decide whether to error or degrade gracefully.

## Decision

Pagination errors log a warning and return the original unpaginated body. The response is not broken by a pagination misconfiguration.

## Consequences

- Responses continue to work even if pagination configuration doesn't match the body structure.
- Easier debugging -- the warning log indicates what went wrong without blocking the response.
- Users might not immediately notice pagination isn't working if they don't check logs.
