# Faker Integration Design

## Goal

Add support for generating realistic fake data in mock responses using [jaswdr/faker](https://github.com/jaswdr/faker) v2. Faker functions are exposed as nested namespaces in templates: `faker.person.name()`, `faker.address.postCode()`, etc.

## API Surface

### Template Syntax

**Expr engine:**
```
${faker.person.name()}
${faker.internet.email()}
${faker.address.city()}
```

**Jinja2 engine:**
```
{{faker.person.name()}}
{{faker.internet.email()}}
{{faker.address.city()}}
```

### Seeding (Optional)

Scenarios can specify a `faker_seed` for deterministic output:

```yaml
response:
  engine: expr
  faker_seed: 42
  body: |
    {"name": "${faker.person.name()}"}
```

When `faker_seed` is set, the same request always produces the same fake data. When absent, each request generates fresh random data.

## Architecture

### New Package: `internal/infrastructure/outbound/faker/`

- **`faker.go`** - `FakerContext` top-level struct with all category fields
- **`categories.go`** - Wrapper structs for each faker category
- **`factory.go`** - `NewFakerContext(seed *int64) FakerContext` factory

### Wrapper Pattern

Each faker category gets a Go struct wrapping the underlying faker methods:

```go
type FakerPerson struct {
    gen *faker.Faker
}

func (p FakerPerson) Name() string      { return p.gen.Person().Name() }
func (p FakerPerson) FirstName() string  { return p.gen.Person().FirstName() }
func (p FakerPerson) LastName() string   { return p.gen.Person().LastName() }
// ... all Person methods

type FakerContext struct {
    Person      FakerPerson
    Address     FakerAddress
    Internet    FakerInternet
    Company     FakerCompany
    // ... all categories
}
```

### Integration Points

| File | Change |
|------|--------|
| `template/functions.go` | Add `FakerContext` to `RenderContext` |
| `template/expr.go` | Register `faker` in Expr environment |
| `template/jinja2.go` | Register `faker` in Pongo2 context |
| `filesystem/yaml_types.go` | Add `FakerSeed *int64` to response YAML |
| `services/compiler.go` | Pass seed through to template compilation |

## Categories

All 39 faker categories exposed:

| Namespace | Key Methods |
|-----------|------------|
| `faker.person` | `name()`, `firstName()`, `lastName()`, `ssn()`, `gender()` |
| `faker.address` | `address()`, `city()`, `state()`, `postCode()`, `country()`, `latitude()`, `longitude()` |
| `faker.internet` | `email()`, `url()`, `ipv4()`, `ipv6()`, `domain()`, `password()`, `macAddress()` |
| `faker.company` | `name()`, `jobTitle()`, `catchPhrase()`, `bs()`, `ein()` |
| `faker.lorem` | `word()`, `sentence()`, `paragraph()` |
| `faker.payment` | `creditCardNumber()`, `creditCardType()`, `iban()` |
| `faker.currency` | `currency()`, `code()` |
| `faker.phone` | `number()`, `e164Number()`, `areaCode()` |
| `faker.hash` | `md5()`, `sha256()`, `sha512()` |
| `faker.uuid` | `v4()` |
| `faker.userAgent` | `userAgent()`, `chrome()`, `firefox()` |
| `faker.color` | `hex()`, `colorName()`, `safeColorName()`, `rgb()` |
| `faker.time` | `dayOfWeek()`, `month()`, `monthName()`, `year()`, `timezone()` |
| `faker.car` | `maker()`, `model()`, `plate()`, `fuelType()` |
| `faker.food` | `fruit()`, `vegetable()` |
| `faker.beer` | `name()`, `style()`, `hop()`, `malt()` |
| `faker.music` | `name()`, `genre()`, `author()` |
| `faker.pet` | `name()`, `cat()`, `dog()` |
| `faker.app` | `name()`, `version()` |
| `faker.blood` | `name()` |
| `faker.boolean` | `bool()` |
| `faker.crypto` | `bitcoinAddress()`, `etheriumAddress()` |
| `faker.emoji` | `emoji()`, `emojiCode()` |
| `faker.file` | `extension()`, `filenameWithExtension()` |
| `faker.gamer` | `tag()` |
| `faker.gender` | `name()`, `abbr()` |
| `faker.genre` | `name()` |
| `faker.language` | `language()`, `programmingLanguage()` |
| `faker.mimeType` | `mimeType()` |
| `faker.pokemon` | `english()`, `japanese()` |
| `faker.youTube` | `generateVideoID()`, `generateFullURL()` |

## Data Flow

1. YAML parsed -> `FakerSeed` extracted from response config
2. Template compilation -> seed passed to `NewFakerContext()`
3. Request handling:
   - Seeded: same faker instance reused per scenario (deterministic)
   - Unseeded: new faker instance per request (random)
4. Template rendering: `FakerContext` added to render environment

## Testing

- Unit tests for each category wrapper (non-empty return values)
- Seeded determinism test (same seed = same output)
- Integration test with scenario YAML using faker in both Expr and Jinja2
