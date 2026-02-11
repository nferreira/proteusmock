# ADR-0020: Include Depth Limit

## Status

Accepted

## Context

The `!include` directive allows YAML files to include other YAML files. Circular includes (A includes B, B includes A) could cause infinite recursion and crash the process.

## Decision

Set a maximum include depth of 10 levels. Includes beyond this depth result in an error.

## Consequences

- Prevents infinite recursion from circular includes.
- Depth of 10 is generous for any practical use case.
- Clear error message when the limit is exceeded.
