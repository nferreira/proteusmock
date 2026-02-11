# ADR-0012: Pongo2 for Jinja2

## Status

Accepted

## Context

The project needs a Go library that implements Jinja2/Django template syntax for the Jinja2 engine option.

## Decision

Use `flosch/pongo2/v6` as the Jinja2 implementation. It is a mature Go library with Django/Jinja2 syntax support.

## Consequences

- Feature-rich template engine with familiar syntax for Python/Django developers.
- Caveat: Pongo2 HTML-escapes output by default, so raw body embedding via `{{ body }}` produces escaped output for JSON. Users should use `jsonPath()` instead.
- Well-maintained library with active community.
