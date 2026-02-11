# Documentation Diff Summary

Generated: 2026-02-11

## Files Written

| File | Action | Top Changes |
|---|---|---|
| `README.md` | Created | Project overview, quickstart, make targets table, docs links |
| `docs/ARCHITECTURE.md` | Created | ASCII component diagram, package map table, request/load flows, ports & adapters table, concurrency model, security notes |
| `docs/USAGE.md` | Created | CLI flags table, admin API reference, full scenario YAML format, string matchers, body matching (JSONPath/XPath/combinators), both template engines with function tables, error responses, 3 golden path examples |
| `docs/EXTENDING.md` | Created | End-to-end feature guide, adding template engines, admin endpoints, repositories, ports; patterns table, testing strategy with fakes table |
| `docs/DECISIONS.md` | Created | Architecture, matching, template, hot-reload, rate-limiting, security, observability decisions; non-goals table |
| `docs/DIFF_SUMMARY.md` | Created | This file |

## Skipped / Unknown Items

| Item | Reason |
|---|---|
| ADR files | None found in repo -- decisions derived from code structure |
| CI/CD config | No CI config files found (`.github/workflows/`, `.gitlab-ci.yml`, etc.) |
| Docker config | No `Dockerfile` or `docker-compose.yml` found |
| LICENSE | Not found in repo |
| CHANGELOG | Not found in repo |
| Environment variables | None used -- all config via CLI flags |
| gRPC / protobuf | Not applicable -- HTTP-only server |
| Database / migrations | Not applicable -- YAML-file-based storage |

## Verification

All documented features verified against source code in:
- `cmd/proteusmock/main.go`
- `internal/app/`, `internal/domain/`, `internal/infrastructure/`
- `mock/scenarios/`, `testdata/`
- `go.mod`, `Makefile`, `scripts/showcase.sh`
