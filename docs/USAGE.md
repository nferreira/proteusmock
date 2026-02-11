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
  pagination:
    style: page_size             # "page_size" (default) or "offset_limit"
    page_param: page             # query param name for page number
    size_param: page_size        # query param name for page size
    default_size: 10             # size when query param is absent
    max_size: 100                # upper bound for size
    data_path: "$"               # JSONPath to the array to paginate
    envelope:                    # customize response wrapper field names
      data_field: data
      page_field: page
      size_field: size
      total_items_field: total_items
      total_pages_field: total_pages
      has_next_field: has_next
      has_previous_field: has_previous
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

## Body Conditions

Body conditions match against the request body using JSONPath (for JSON) or XPath (for XML) extractors. Conditions can be combined with boolean combinators.

### Simple condition

```yaml
when:
  method: POST
  path: /api/v1/users
  body:
    content_type: json
    conditions:
      - extractor: "$.name"
        matcher: "=Alice"        # exact match
      - extractor: "$.age"
        matcher: "^\\d{2,}"      # regex: 2+ digit number
```

### OR combinator (`any`)

Matches if **at least one** child clause matches.

```yaml
body:
  content_type: json
  any:
    - conditions:
        - extractor: "$.method"
          matcher: "=credit_card"
    - conditions:
        - extractor: "$.method"
          matcher: "=paypal"
```

### AND combinator (`all`)

Matches if **all** child clauses match.

```yaml
body:
  content_type: json
  all:
    - conditions:
        - extractor: "$.status"
          matcher: "=confirmed"
    - conditions:
        - extractor: "$.total"
          matcher: "^\\d{3,}"
```

### NOT combinator (`not`)

Matches if the child clause does **not** match.

```yaml
body:
  content_type: json
  not:
    conditions:
      - extractor: "$.role"
        matcher: "=banned"
```

### Nested combinators

Combinators compose arbitrarily. Example: active AND (admin OR moderator) AND NOT suspended.

```yaml
body:
  content_type: json
  all:
    - conditions:
        - extractor: "$.status"
          matcher: "=active"
    - any:
        - conditions:
            - extractor: "$.role"
              matcher: "=admin"
        - conditions:
            - extractor: "$.role"
              matcher: "=moderator"
    - not:
        conditions:
          - extractor: "$.suspended"
            matcher: "=true"
```

## Pagination

ProteusMock can automatically paginate JSON array responses. Define the full dataset in your response body and configure pagination under `policy.pagination` -- the server slices the array and wraps it in a pagination envelope at request time.

Pagination is a **post-rendering** step: the response body is rendered first (including templates), then parsed as JSON, sliced, and wrapped.

### Pagination styles

| Style | Default params | Description |
|---|---|---|
| `page_size` (default) | `?page=1&size=10` | 1-based page number + page size |
| `offset_limit` | `?offset=0&limit=10` | 0-based offset + limit |

### Configuration reference

```yaml
policy:
  pagination:
    style: page_size          # "page_size" or "offset_limit"
    page_param: page          # query param for page number (page_size style)
    size_param: page_size     # query param for page size (page_size style)
    offset_param: offset      # query param for offset (offset_limit style)
    limit_param: limit        # query param for limit (offset_limit style)
    default_size: 10          # default items per page when param is absent
    max_size: 100             # upper bound — requests above this are clamped
    data_path: "$"            # JSONPath to the array to paginate
    envelope:                 # customize response wrapper field names
      data_field: data
      page_field: page
      size_field: size
      total_items_field: total_items
      total_pages_field: total_pages
      has_next_field: has_next
      has_previous_field: has_previous
```

All fields are optional. Omitted fields use the defaults shown above.

### Defaults

| Field | Default |
|---|---|
| `style` | `page_size` |
| `page_param` | `page` |
| `size_param` | `size` |
| `offset_param` | `offset` |
| `limit_param` | `limit` |
| `default_size` | `10` |
| `max_size` | `100` |
| `data_path` | `$` (root array) |

### Example: page + page_size

```yaml
id: paginated-users
name: Paginated users
priority: 10
when:
  method: GET
  path: /api/v1/paginated/users
policy:
  pagination:
    style: page_size
    page_param: page
    size_param: page_size
    default_size: 5
    max_size: 20
    data_path: "$.users"
response:
  status: 200
  headers:
    Content-Type: application/json
  body: |
    {
      "users": [
        {"id": 1, "name": "Alice"},
        {"id": 2, "name": "Bob"},
        {"id": 3, "name": "Charlie"},
        {"id": 4, "name": "Diana"},
        {"id": 5, "name": "Eve"}
      ]
    }
```

```bash
# First page, default size (5 items)
curl http://localhost:8080/api/v1/paginated/users

# Page 1 with 2 items per page
curl "http://localhost:8080/api/v1/paginated/users?page=1&page_size=2"
```

Response for `?page=1&page_size=2`:

```json
{
  "data": [
    {"id": 1, "name": "Alice"},
    {"id": 2, "name": "Bob"}
  ],
  "page": 1,
  "size": 2,
  "total_items": 5,
  "total_pages": 3,
  "has_next": true,
  "has_previous": false
}
```

### Example: offset + limit with custom envelope

```yaml
id: paginated-products
name: Paginated products
priority: 10
when:
  method: GET
  path: /api/v1/paginated/products
policy:
  pagination:
    style: offset_limit
    offset_param: offset
    limit_param: limit
    default_size: 5
    data_path: "$.products"
    envelope:
      data_field: results
      total_items_field: count
response:
  status: 200
  headers:
    Content-Type: application/json
  body: |
    {
      "products": [
        {"sku": "PROD-1", "name": "Widget", "price": 9.99},
        {"sku": "PROD-2", "name": "Gadget", "price": 19.99},
        {"sku": "PROD-3", "name": "Doohickey", "price": 4.99}
      ]
    }
```

```bash
curl "http://localhost:8080/api/v1/paginated/products?offset=0&limit=2"
```

Response:

```json
{
  "results": [
    {"sku": "PROD-1", "name": "Widget", "price": 9.99},
    {"sku": "PROD-2", "name": "Gadget", "price": 19.99}
  ],
  "count": 3,
  "page": 1,
  "size": 2,
  "total_pages": 2,
  "has_next": true,
  "has_previous": false
}
```

### Pagination with templates

Pagination works with template engines. The body is rendered first, then paginated:

```yaml
id: dynamic-paginated
name: Dynamic paginated catalog
priority: 10
when:
  method: GET
  path: /api/v1/paginated/catalog
policy:
  pagination:
    default_size: 5
    data_path: "$.items"
response:
  status: 200
  engine: expr
  headers:
    Content-Type: application/json
  body: |
    {
      "items": ${toJSON(seq(1, 20))}
    }
```

```bash
curl "http://localhost:8080/api/v1/paginated/catalog?page=1&size=5"
```

### Graceful degradation

If pagination fails (e.g., the `data_path` doesn't point to an array, or the body isn't valid JSON), the server logs a warning and returns the original unpaginated response body.

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

### 4. Body matching with OR combinator

```yaml
id: payment
name: Accept Payment
priority: 10
when:
  method: POST
  path: /api/v1/payments
  body:
    content_type: json
    any:
      - conditions:
          - extractor: "$.method"
            matcher: "=credit_card"
      - conditions:
          - extractor: "$.method"
            matcher: "=paypal"
response:
  status: 200
  engine: expr
  body: |
    {"status": "accepted", "method": "${jsonPath('$.method')}"}
```

```bash
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{"method": "credit_card", "amount": 49.99}'
```

### 5. Paginated endpoint

```yaml
id: users-list
name: Users List
priority: 10
when:
  method: GET
  path: /api/v1/users
policy:
  pagination:
    size_param: page_size
    default_size: 2
response:
  status: 200
  headers:
    Content-Type: application/json
  body: |
    [
      {"id": 1, "name": "Alice"},
      {"id": 2, "name": "Bob"},
      {"id": 3, "name": "Charlie"},
      {"id": 4, "name": "Diana"},
      {"id": 5, "name": "Eve"}
    ]
```

```bash
curl "http://localhost:8080/api/v1/users?page=1&page_size=2"
# Returns: {"data":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}],"page":1,"size":2,"total_items":5,"total_pages":3,"has_next":true,"has_previous":false}

curl "http://localhost:8080/api/v1/users?page=3&page_size=2"
# Returns: {"data":[{"id":5,"name":"Eve"}],"page":3,"size":2,"total_items":5,"total_pages":3,"has_next":false,"has_previous":true}
```
