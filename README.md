<p align="center">
  <img src="assets/logo.png" alt="ProteusMock Logo" width="400">
</p>

# ProteusMock

A programmable HTTP mock server for API development and testing. Define request-matching rules and dynamic responses in YAML, get instant hot-reload on file changes.

## Why "Proteus"?

In Greek mythology, [Proteus](https://en.wikipedia.org/wiki/Proteus) was a sea god who could shift into any form at will. ProteusMock embodies this idea: a single server that reshapes itself into whatever API your tests demand -- different endpoints, dynamic responses, conditional logic -- all from declarative YAML.

## Features

- **Declarative YAML scenarios** with method, path, header, and body matching
- **Body matching** via JSONPath / XPath with boolean combinators (`all`, `any`, `not`)
- **Dynamic responses** using Expr (`${ }`) or Jinja2 (`{{ }}`) template engines
- **Automatic pagination** -- page+size or offset+limit with customizable params and envelope
- **Hot reload** -- edit YAML files and the server picks up changes automatically
- **Rate limiting** per scenario with token-bucket algorithm
- **Latency simulation** with fixed delay + jitter
- **Admin API** for inspecting loaded scenarios and request traces
- **Web dashboard** -- built-in SPA at `/__ui/` for browsing, editing, and creating scenarios
- **`!include`** directive for reusable YAML fragments and response files

## Quickstart

```bash
# Build
make build

# Run with default mock directory
make run
# or
bin/proteusmock --root ./mock --port 8080

# Test
curl http://localhost:8080/api/v1/health
```

## Docker

```bash
# Build the image (runs unit tests during build)
make docker-build

# Run with bundled mock scenarios
make docker-run

# Override scenarios with a volume mount
docker run --rm -p 8080:8080 -v ./my-mocks:/mock:ro sophialabs/proteusmock:latest

# Build running all tests (unit + integration + E2E)
make docker-test

# Skip tests during build
docker build --build-arg RUN_TESTS=false -t sophialabs/proteusmock:latest .

# Using docker compose
make compose-up     # build + start in background
make compose-down   # stop
```

## Minimal scenario

```yaml
# mock/scenarios/health.yaml
id: health-check
name: Health Check
priority: 100
when:
  method: GET
  path: /api/v1/health
response:
  status: 200
  headers:
    Content-Type: application/json
  body: '{"status": "ok"}'
```

## Documentation

| Document | Contents |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design, package map, data flows |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | Tutorial: scenarios, matching, templates, pagination |
| [docs/USAGE.md](docs/USAGE.md) | CLI flags, API reference, scenario format, examples |
| [docs/EXTENDING.md](docs/EXTENDING.md) | Adding endpoints, engines, storage backends, testing |
| [docs/DECISIONS.md](docs/DECISIONS.md) | Key design decisions and trade-offs |

## Make targets

```
make help
```

| Target | Description |
|---|---|
| `build` | Build binary to `bin/proteusmock` |
| `run` | Run server (`--root ./mock --port 8080`) |
| `test` | Run unit tests only (fast) |
| `test-integration` | Run unit + integration tests |
| `test-e2e` | Run E2E tests only |
| `test-all` | Run all tests (unit + integration + E2E) with race detector |
| `test-race` | Run unit tests with race detector |
| `test-cover` | Run unit + integration tests with coverage report |
| `fmt` | Format with gofmt |
| `vet` | Run go vet |
| `lint` | Run staticcheck |
| `showcase` | Start server and run all demo scenarios |
| `docker-build` | Build Docker production image |
| `docker-run` | Run Docker container (port 8080, bundled mocks) |
| `docker-test` | Build Docker image running all tests |
| `docker-push` | Push image to registry |
| `compose-up` | Start services with docker compose |
| `compose-down` | Stop docker compose services |
| `clean` | Remove build artifacts |

## CI

GitHub Actions runs on every push to `main` and on pull requests:

| Job | What it does |
|-----|--------------|
| **Lint** | `gofmt` check, `go vet`, `staticcheck` |
| **Unit Tests** | `go test -race` with coverage report |
| **Integration Tests** | `go test -tags=integration -race` |
| **E2E Tests** | `go test -tags=e2e -race` |
| **Build** | Static binary compilation, uploaded as artifact |
| **Docker** | Builds image, verifies health endpoint (runs after all test jobs pass) |

## Dependencies

Go 1.25+, [chi/v5](https://github.com/go-chi/chi), [expr](https://github.com/expr-lang/expr), [pongo2](https://github.com/flosch/pongo2), [fsnotify](https://github.com/fsnotify/fsnotify), [yaml.v3](https://gopkg.in/yaml.v3).

## License

See [LICENSE](LICENSE.md).
