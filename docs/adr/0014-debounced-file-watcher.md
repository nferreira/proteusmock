# ADR-0014: Debounced File Watcher

## Status

Accepted

## Context

File system watchers (fsnotify) emit multiple events when editors save files (e.g., write temp file, rename, chmod). Without debouncing, a single save could trigger multiple reloads.

## Decision

Debounce `fsnotify` events to 500ms. Multiple events within the window are collapsed into a single reload.

## Consequences

- Prevents rapid, redundant recompilation during editor saves.
- 500ms delay before reload is imperceptible for development workflows.
- Simple implementation with a timer reset on each event.
