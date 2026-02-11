# ADR-0019: Symlink Path Traversal Prevention

## Status

Accepted

## Context

The `!include` directive and `body_file` field allow referencing external files. Without proper validation, symlinks or `../` sequences could be used to read files outside the intended root directory.

## Decision

Use `filepath.EvalSymlinks` to resolve the real path, then verify with a prefix check that the resolved path stays within the `--root` directory.

## Consequences

- Prevents path traversal attacks via symlinks and relative paths.
- Users cannot accidentally or intentionally reference files outside the scenario root.
- Slightly more I/O per include resolution due to symlink evaluation.
