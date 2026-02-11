# Usage

## Quickstart

```bash
make build
bin/proteusmock --root ./mock --port 8080
curl http://localhost:8080/api/v1/health
```

## CLI Flags

| Flag | Default | Description |
|---|---|---|
| `--root` | `./mock` | Root directory for scenario YAML files |
| `--port` | `8080` | HTTP listen port |
| `--trace-size` | `200` | Trace ring buffer capacity |
| `--log-level` | `debug` | `debug`, `info`, `warn`, `error` |
| `--default-engine` | *(empty)* | Default template engine: `expr` or `jinja2` |

## Admin API

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/__admin/scenarios` | List all loaded scenarios |
| `GET` | `/__admin/scenarios/search?q=<term>` | Search by ID, name, or path |
| `GET` | `/__admin/trace?last=<n>` | Last *n* trace entries (default 10) |
| `POST` | `/__admin/reload` | Force scenario reload |

```bash
curl -s http://localhost:8080/__admin/scenarios | jq .
curl -s 'http://localhost:8080/__admin/trace?last=5' | jq .
```

## Scenario YAML Format

### Minimal

```yaml
id: health-check
name: Health Check
priority: 100
when:
  method: GET
  path: /api/v1/health
response:
  status: 200
  body: '{"status": "ok"}'
```

### Full reference

```yaml
id: unique-id                   # required, must be unique
name: Human-readable name       # required
priority: 10                    # higher = matched first

when:
  method: POST
  path: /api/v1/users/{id}     # chi-style path params
  headers:
    Content-Type: =application/json    # "=" -> exact, otherwise regex
    Authorization: "Bearer .*"
  body:
    content_type: json          # "json" or "xml"
    conditions:
      - extractor: "$.user.name"       # JSONPath or XPath
        matcher: "=Alice"
    all: [...]                  # AND (recursive)
    any: [...]                  # OR  (recursive)
    not: { ... }                # NOT (recursive)

response:
  status: 200
  headers: { Content-Type: application/json }
  body: '{"inline": true}'             # or body_file: responses/data.json
  engine: expr                         # "expr" or "jinja2" for templates
  content_type: application/json       # optional, auto-inferred

policy:
  rate_limit: { rate: 10.0, burst: 20, key: my-key }
  latency: { fixed_ms: 100, jitter_ms: 50 }
```

Multiple scenarios per file: use a YAML list (`- id: ...`).

### `!include` directive

```yaml
body: !include @root/responses/data.json    # relative to --root
body: !include @here/sibling.json           # relative to current file
body: !include ../shared/fragment.yaml      # relative to current file
```

- `.yaml`/`.yml` files: parsed and recursively resolved (max depth 10)
- Other files: inserted as raw strings
- Path traversal outside `--root` is rejected

## String Matchers

| Syntax | Meaning | Example |
|---|---|---|
| `=value` | Exact match | `=application/json` |
| `pattern` | Regex | `Bearer .*` |

## Template Engines

### Expr (`engine: expr`) -- `${ expression }`

```yaml
response:
  engine: expr
  body: |
    {
      "user_id": "${pathParam('id')}",
      "name": "${jsonPath('$.user.name')}",
      "uuid": "${uuid()}",
      "lucky": ${randomInt(1, 100)},
      "items": ${toJSON(seq(1, 5))}
    }
```

### Jinja2 (`engine: jinja2`) -- `{{ var }}`, `{% if %}`, `{% for %}`

```yaml
response:
  engine: jinja2
  body: |
    {
      "source": "{{ queryParam('source') }}",
      {% if header('X-Tier') == 'premium' %}
      "tier": "premium", "limit": 1000
      {% else %}
      "tier": "free", "limit": 10
      {% endif %}
    }
```

### Functions (both engines)

| Function | Description |
|---|---|
| `pathParam(name)` | Path parameter value |
| `queryParam(name)` | Query parameter value |
| `header(name)` | Header value (case-insensitive) |
| `body()` | Raw request body (Expr only) |
| `now()` | ISO-8601 timestamp |
| `nowFormat(layout)` | Go-formatted timestamp |
| `uuid()` | Random UUID v4 |
| `randomInt(min, max)` | Random int in [min, max] |
| `seq(start, end)` | Integer sequence |
| `toJSON(value)` | Marshal to JSON |
| `jsonPath(expr)` | Extract from request body |

Jinja2 also exposes variables: `method`, `path`, `headers`, `queryParams`, `pathParams`, `body`, `now`.

## Error Responses

| Code | Condition | Notes |
|---|---|---|
| 404 | No route or no predicate matched | Includes `candidates` with failure details |
| 429 | Rate limited | `Retry-After: 1` header |
| 503 | Server not ready | Index not yet loaded |

## Examples

### 1. Static endpoint with path params

```yaml
id: get-user
name: Get User
priority: 10
when: { method: GET, path: /api/v1/users/{id} }
response:
  status: 200
  body: '{"id": 1, "name": "Alice"}'
```

```bash
curl http://localhost:8080/api/v1/users/42
```

### 2. Dynamic echo with Expr

```yaml
id: echo
name: Echo Body
priority: 10
when: { method: POST, path: /api/v1/echo }
response:
  status: 200
  engine: expr
  body: |
    {"name": "${jsonPath('$.user.name')}", "id": "${uuid()}"}
```

```bash
curl -X POST http://localhost:8080/api/v1/echo \
  -H "Content-Type: application/json" \
  -d '{"user": {"name": "Alice"}}'
```

### 3. Conditional with Jinja2

```yaml
id: flags
name: Feature Flags
priority: 10
when: { method: GET, path: /api/v1/flags }
response:
  status: 200
  engine: jinja2
  body: |
    {% if header('X-Tier') == 'premium' %}
    {"tier": "premium", "limit": 1000}
    {% else %}
    {"tier": "free", "limit": 10}
    {% endif %}
```

```bash
curl http://localhost:8080/api/v1/flags -H "X-Tier: premium"
curl http://localhost:8080/api/v1/flags
```
