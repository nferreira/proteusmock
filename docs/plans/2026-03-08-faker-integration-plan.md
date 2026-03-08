# Faker Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `jaswdr/faker` v2 support so scenarios can generate realistic fake data via `faker.person.name()`, `faker.address.city()`, etc. in both Expr and Jinja2 template engines, with optional seeding for deterministic output.

**Architecture:** Create a `faker` wrapper package with Go structs mirroring faker's category hierarchy. Add `FakerSeed *int64` flowing from YAML through domain/compiled types to `RenderContext`. At render time, build a `FakerContext` from the seed and inject it into both template engines.

**Tech Stack:** Go, jaswdr/faker v2, expr-lang/expr, flosch/pongo2

---

### Task 1: Add faker dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the dependency**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go get github.com/jaswdr/faker/v2`

Expected: go.mod updated with `github.com/jaswdr/faker/v2`

**Step 2: Verify**

Run: `grep jaswdr go.mod`

Expected: `github.com/jaswdr/faker/v2 v2.x.x`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add jaswdr/faker v2 dependency"
```

---

### Task 2: Create faker wrapper package — core categories

**Files:**
- Create: `internal/infrastructure/outbound/faker/faker.go`
- Test: `internal/infrastructure/outbound/faker/faker_test.go`

**Step 1: Write the failing test**

Create `internal/infrastructure/outbound/faker/faker_test.go`:

```go
package faker_test

import (
	"testing"

	fakerpkg "github.com/sophialabs/proteusmock/internal/infrastructure/outbound/faker"
)

func TestNewFakerContext_Unseeded(t *testing.T) {
	ctx := fakerpkg.NewFakerContext(nil)

	// Person
	if name := ctx.Person.Name(); name == "" {
		t.Error("expected non-empty person name")
	}
	if first := ctx.Person.FirstName(); first == "" {
		t.Error("expected non-empty first name")
	}
	if last := ctx.Person.LastName(); last == "" {
		t.Error("expected non-empty last name")
	}

	// Address
	if city := ctx.Address.City(); city == "" {
		t.Error("expected non-empty city")
	}
	if country := ctx.Address.Country(); country == "" {
		t.Error("expected non-empty country")
	}
	if postCode := ctx.Address.PostCode(); postCode == "" {
		t.Error("expected non-empty post code")
	}

	// Internet
	if email := ctx.Internet.Email(); email == "" {
		t.Error("expected non-empty email")
	}
	if url := ctx.Internet.URL(); url == "" {
		t.Error("expected non-empty URL")
	}
	if ip := ctx.Internet.Ipv4(); ip == "" {
		t.Error("expected non-empty IPv4")
	}

	// Company
	if name := ctx.Company.Name(); name == "" {
		t.Error("expected non-empty company name")
	}
	if title := ctx.Company.JobTitle(); title == "" {
		t.Error("expected non-empty job title")
	}
}

func TestNewFakerContext_Seeded_Deterministic(t *testing.T) {
	seed := int64(42)

	ctx1 := fakerpkg.NewFakerContext(&seed)
	ctx2 := fakerpkg.NewFakerContext(&seed)

	if ctx1.Person.Name() != ctx2.Person.Name() {
		t.Error("seeded faker should produce same person name")
	}
	if ctx1.Address.City() != ctx2.Address.City() {
		t.Error("seeded faker should produce same city")
	}
	if ctx1.Internet.Email() != ctx2.Internet.Email() {
		t.Error("seeded faker should produce same email")
	}
}

func TestNewFakerContext_AllCategories(t *testing.T) {
	ctx := fakerpkg.NewFakerContext(nil)

	// Verify all category fields are accessible and return non-empty values
	checks := []struct {
		name string
		fn   func() string
	}{
		{"person.name", ctx.Person.Name},
		{"address.city", ctx.Address.City},
		{"internet.email", ctx.Internet.Email},
		{"company.name", ctx.Company.Name},
		{"lorem.word", ctx.Lorem.Word},
		{"lorem.sentence", ctx.Lorem.Sentence},
		{"payment.creditCardNumber", ctx.Payment.CreditCardNumber},
		{"payment.creditCardType", ctx.Payment.CreditCardType},
		{"currency.Currency", ctx.Currency.Currency},
		{"currency.code", ctx.Currency.Code},
		{"phone.number", ctx.Phone.Number},
		{"hash.md5", ctx.Hash.MD5},
		{"hash.sha256", ctx.Hash.SHA256},
		{"uuid.v4", ctx.UUID.V4},
		{"userAgent.userAgent", ctx.UserAgent.UserAgent},
		{"color.hex", ctx.Color.Hex},
		{"color.colorName", ctx.Color.ColorName},
		{"car.maker", ctx.Car.Maker},
		{"car.model", ctx.Car.Model},
		{"food.fruit", ctx.Food.Fruit},
		{"food.vegetable", ctx.Food.Vegetable},
		{"beer.name", ctx.Beer.Name},
		{"beer.style", ctx.Beer.Style},
		{"music.name", ctx.Music.Name},
		{"music.genre", ctx.Music.Genre},
		{"pet.name", ctx.Pet.Name},
		{"pet.cat", ctx.Pet.Cat},
		{"pet.dog", ctx.Pet.Dog},
		{"app.name", ctx.App.Name},
		{"app.version", ctx.App.Version},
		{"blood.name", ctx.Blood.Name},
		{"crypto.BitcoinAddress", ctx.Crypto.BitcoinAddress},
		{"emoji.emoji", ctx.Emoji.Emoji},
		{"file.extension", ctx.File.Extension},
		{"gamer.tag", ctx.Gamer.Tag},
		{"gender.name", ctx.Gender.Name},
		{"genre.name", ctx.Genre.Name},
		{"language.language", ctx.Language.Language},
		{"language.programmingLanguage", ctx.Language.ProgrammingLanguage},
		{"mimeType.mimeType", ctx.MimeType.MimeType},
		{"pokemon.english", ctx.Pokemon.English},
		{"youTube.generateVideoID", ctx.YouTube.GenerateVideoID},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			val := check.fn()
			if val == "" {
				t.Errorf("%s returned empty string", check.name)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go test ./internal/infrastructure/outbound/faker/...`

Expected: FAIL — package does not exist

**Step 3: Write the implementation**

Create `internal/infrastructure/outbound/faker/faker.go`:

```go
package faker

import (
	"math/rand/v2"

	"github.com/jaswdr/faker/v2"
)

// FakerContext provides realistic fake data generation organized by category.
// All field names use PascalCase for Go export, but template engines access them
// as lowercase (e.g., faker.person.name() in Expr, faker.person.name() in Jinja2).
type FakerContext struct {
	Person    FakerPerson
	Address   FakerAddress
	Internet  FakerInternet
	Company   FakerCompany
	Lorem     FakerLorem
	Payment   FakerPayment
	Currency  FakerCurrency
	Phone     FakerPhone
	Hash      FakerHash
	UUID      FakerUUID
	UserAgent FakerUserAgent
	Color     FakerColor
	Time      FakerTime
	Car       FakerCar
	Food      FakerFood
	Beer      FakerBeer
	Music     FakerMusic
	Pet       FakerPet
	App       FakerApp
	Blood     FakerBlood
	Boolean   FakerBoolean
	Crypto    FakerCrypto
	Emoji     FakerEmoji
	File      FakerFile
	Gamer     FakerGamer
	Gender    FakerGender
	Genre     FakerGenre
	Language  FakerLanguage
	MimeType  FakerMimeType
	Pokemon   FakerPokemon
	YouTube   FakerYouTube
}

// NewFakerContext creates a FakerContext. If seed is non-nil, the faker instance
// is deterministic (same seed = same data). If nil, uses a random source.
func NewFakerContext(seed *int64) FakerContext {
	var gen faker.Faker
	if seed != nil {
		src := rand.NewPCG(uint64(*seed), 0)
		gen = faker.NewWithSeed(src)
	} else {
		gen = faker.New()
	}

	return FakerContext{
		Person:    FakerPerson{gen: gen},
		Address:   FakerAddress{gen: gen},
		Internet:  FakerInternet{gen: gen},
		Company:   FakerCompany{gen: gen},
		Lorem:     FakerLorem{gen: gen},
		Payment:   FakerPayment{gen: gen},
		Currency:  FakerCurrency{gen: gen},
		Phone:     FakerPhone{gen: gen},
		Hash:      FakerHash{gen: gen},
		UUID:      FakerUUID{gen: gen},
		UserAgent: FakerUserAgent{gen: gen},
		Color:     FakerColor{gen: gen},
		Time:      FakerTime{gen: gen},
		Car:       FakerCar{gen: gen},
		Food:      FakerFood{gen: gen},
		Beer:      FakerBeer{gen: gen},
		Music:     FakerMusic{gen: gen},
		Pet:       FakerPet{gen: gen},
		App:       FakerApp{gen: gen},
		Blood:     FakerBlood{gen: gen},
		Boolean:   FakerBoolean{gen: gen},
		Crypto:    FakerCrypto{gen: gen},
		Emoji:     FakerEmoji{gen: gen},
		File:      FakerFile{gen: gen},
		Gamer:     FakerGamer{gen: gen},
		Gender:    FakerGender{gen: gen},
		Genre:     FakerGenre{gen: gen},
		Language:  FakerLanguage{gen: gen},
		MimeType:  FakerMimeType{gen: gen},
		Pokemon:   FakerPokemon{gen: gen},
		YouTube:   FakerYouTube{gen: gen},
	}
}

// --- Category wrappers ---

type FakerPerson struct{ gen faker.Faker }

func (f FakerPerson) Name() string            { return f.gen.Person().Name() }
func (f FakerPerson) FirstName() string       { return f.gen.Person().FirstName() }
func (f FakerPerson) FirstNameMale() string   { return f.gen.Person().FirstNameMale() }
func (f FakerPerson) FirstNameFemale() string { return f.gen.Person().FirstNameFemale() }
func (f FakerPerson) LastName() string        { return f.gen.Person().LastName() }
func (f FakerPerson) Title() string           { return f.gen.Person().Title() }
func (f FakerPerson) TitleMale() string       { return f.gen.Person().TitleMale() }
func (f FakerPerson) TitleFemale() string     { return f.gen.Person().TitleFemale() }
func (f FakerPerson) Suffix() string          { return f.gen.Person().Suffix() }
func (f FakerPerson) Gender() string          { return f.gen.Person().Gender() }
func (f FakerPerson) GenderMale() string      { return f.gen.Person().GenderMale() }
func (f FakerPerson) GenderFemale() string    { return f.gen.Person().GenderFemale() }
func (f FakerPerson) NameMale() string        { return f.gen.Person().NameMale() }
func (f FakerPerson) NameFemale() string      { return f.gen.Person().NameFemale() }
func (f FakerPerson) SSN() string             { return f.gen.Person().SSN() }

type FakerAddress struct{ gen faker.Faker }

func (f FakerAddress) Address() string          { return f.gen.Address().Address() }
func (f FakerAddress) BuildingNumber() string   { return f.gen.Address().BuildingNumber() }
func (f FakerAddress) City() string             { return f.gen.Address().City() }
func (f FakerAddress) CityPrefix() string       { return f.gen.Address().CityPrefix() }
func (f FakerAddress) CitySuffix() string       { return f.gen.Address().CitySuffix() }
func (f FakerAddress) Country() string          { return f.gen.Address().Country() }
func (f FakerAddress) CountryAbbr() string      { return f.gen.Address().CountryAbbr() }
func (f FakerAddress) CountryCode() string      { return f.gen.Address().CountryCode() }
func (f FakerAddress) Latitude() float64        { return f.gen.Address().Latitude() }
func (f FakerAddress) Longitude() float64       { return f.gen.Address().Longitude() }
func (f FakerAddress) PostCode() string         { return f.gen.Address().PostCode() }
func (f FakerAddress) SecondaryAddress() string { return f.gen.Address().SecondaryAddress() }
func (f FakerAddress) State() string            { return f.gen.Address().State() }
func (f FakerAddress) StateAbbr() string        { return f.gen.Address().StateAbbr() }
func (f FakerAddress) StreetAddress() string    { return f.gen.Address().StreetAddress() }
func (f FakerAddress) StreetName() string       { return f.gen.Address().StreetName() }
func (f FakerAddress) StreetSuffix() string     { return f.gen.Address().StreetSuffix() }

type FakerInternet struct{ gen faker.Faker }

func (f FakerInternet) Email() string             { return f.gen.Internet().Email() }
func (f FakerInternet) FreeEmail() string          { return f.gen.Internet().FreeEmail() }
func (f FakerInternet) SafeEmail() string          { return f.gen.Internet().SafeEmail() }
func (f FakerInternet) CompanyEmail() string       { return f.gen.Internet().CompanyEmail() }
func (f FakerInternet) User() string               { return f.gen.Internet().User() }
func (f FakerInternet) Password() string           { return f.gen.Internet().Password() }
func (f FakerInternet) URL() string                { return f.gen.Internet().URL() }
func (f FakerInternet) Domain() string             { return f.gen.Internet().Domain() }
func (f FakerInternet) TLD() string                { return f.gen.Internet().TLD() }
func (f FakerInternet) Slug() string               { return f.gen.Internet().Slug() }
func (f FakerInternet) Ipv4() string               { return f.gen.Internet().Ipv4() }
func (f FakerInternet) Ipv6() string               { return f.gen.Internet().Ipv6() }
func (f FakerInternet) LocalIpv4() string          { return f.gen.Internet().LocalIpv4() }
func (f FakerInternet) MacAddress() string         { return f.gen.Internet().MacAddress() }
func (f FakerInternet) HTTPMethod() string         { return f.gen.Internet().HTTPMethod() }
func (f FakerInternet) StatusCode() int            { return f.gen.Internet().StatusCode() }
func (f FakerInternet) StatusCodeMessage() string  { return f.gen.Internet().StatusCodeMessage() }
func (f FakerInternet) Query() string              { return f.gen.Internet().Query() }

type FakerCompany struct{ gen faker.Faker }

func (f FakerCompany) Name() string        { return f.gen.Company().Name() }
func (f FakerCompany) Suffix() string      { return f.gen.Company().Suffix() }
func (f FakerCompany) CatchPhrase() string { return f.gen.Company().CatchPhrase() }
func (f FakerCompany) BS() string          { return f.gen.Company().BS() }
func (f FakerCompany) JobTitle() string    { return f.gen.Company().JobTitle() }
func (f FakerCompany) EIN() string         { return f.gen.Company().EIN() }

type FakerLorem struct{ gen faker.Faker }

func (f FakerLorem) Word() string      { return f.gen.Lorem().Word() }
func (f FakerLorem) Sentence() string  { return f.gen.Lorem().Sentence(6) }
func (f FakerLorem) Paragraph() string { return f.gen.Lorem().Paragraph(3) }

type FakerPayment struct{ gen faker.Faker }

func (f FakerPayment) CreditCardNumber() string             { return f.gen.Payment().CreditCardNumber() }
func (f FakerPayment) CreditCardType() string               { return f.gen.Payment().CreditCardType() }
func (f FakerPayment) CreditCardExpirationDateString() string { return f.gen.Payment().CreditCardExpirationDateString() }
func (f FakerPayment) Iban() string                          { return f.gen.Payment().Iban() }

type FakerCurrency struct{ gen faker.Faker }

func (f FakerCurrency) Currency() string { return f.gen.Currency().Currency() }
func (f FakerCurrency) Code() string     { return f.gen.Currency().Code() }
func (f FakerCurrency) Number() string   { return f.gen.Currency().Number() }
func (f FakerCurrency) Country() string  { return f.gen.Currency().Country() }

type FakerPhone struct{ gen faker.Faker }

func (f FakerPhone) Number() string           { return f.gen.Phone().Number() }
func (f FakerPhone) E164Number() string       { return f.gen.Phone().E164Number() }
func (f FakerPhone) AreaCode() string         { return f.gen.Phone().AreaCode() }
func (f FakerPhone) ExchangeCode() string     { return f.gen.Phone().ExchangeCode() }
func (f FakerPhone) TollFreeAreaCode() string { return f.gen.Phone().TollFreeAreaCode() }
func (f FakerPhone) ToolFreeNumber() string   { return f.gen.Phone().ToolFreeNumber() }

type FakerHash struct{ gen faker.Faker }

func (f FakerHash) MD5() string    { return f.gen.Hash().MD5() }
func (f FakerHash) SHA256() string { return f.gen.Hash().SHA256() }
func (f FakerHash) SHA512() string { return f.gen.Hash().SHA512() }

type FakerUUID struct{ gen faker.Faker }

func (f FakerUUID) V4() string { return f.gen.UUID().V4() }

type FakerUserAgent struct{ gen faker.Faker }

func (f FakerUserAgent) UserAgent() string        { return f.gen.UserAgent().UserAgent() }
func (f FakerUserAgent) Chrome() string            { return f.gen.UserAgent().Chrome() }
func (f FakerUserAgent) Firefox() string           { return f.gen.UserAgent().Firefox() }
func (f FakerUserAgent) Safari() string            { return f.gen.UserAgent().Safari() }
func (f FakerUserAgent) InternetExplorer() string  { return f.gen.UserAgent().InternetExplorer() }
func (f FakerUserAgent) Opera() string             { return f.gen.UserAgent().Opera() }

type FakerColor struct{ gen faker.Faker }

func (f FakerColor) Hex() string           { return f.gen.Color().Hex() }
func (f FakerColor) RGB() string           { return f.gen.Color().RGB() }
func (f FakerColor) ColorName() string     { return f.gen.Color().ColorName() }
func (f FakerColor) SafeColorName() string { return f.gen.Color().SafeColorName() }
func (f FakerColor) CSS() string           { return f.gen.Color().CSS() }

type FakerTime struct{ gen faker.Faker }

func (f FakerTime) DayOfWeek() string  { return f.gen.Time().DayOfWeek() }
func (f FakerTime) DayOfMonth() string { return f.gen.Time().DayOfMonth() }
func (f FakerTime) Month() string      { return f.gen.Time().Month() }
func (f FakerTime) MonthName() string  { return f.gen.Time().MonthName() }
func (f FakerTime) Year() string       { return f.gen.Time().Year() }
func (f FakerTime) Century() string    { return f.gen.Time().Century() }
func (f FakerTime) Timezone() string   { return f.gen.Time().Timezone() }
func (f FakerTime) AmPm() string       { return f.gen.Time().AmPm() }

type FakerCar struct{ gen faker.Faker }

func (f FakerCar) Maker() string            { return f.gen.Car().Maker() }
func (f FakerCar) Model() string            { return f.gen.Car().Model() }
func (f FakerCar) Category() string         { return f.gen.Car().Category() }
func (f FakerCar) FuelType() string         { return f.gen.Car().FuelType() }
func (f FakerCar) TransmissionGear() string { return f.gen.Car().TransmissionGear() }
func (f FakerCar) Plate() string            { return f.gen.Car().Plate() }

type FakerFood struct{ gen faker.Faker }

func (f FakerFood) Fruit() string     { return f.gen.Food().Fruit() }
func (f FakerFood) Vegetable() string { return f.gen.Food().Vegetable() }

type FakerBeer struct{ gen faker.Faker }

func (f FakerBeer) Name() string  { return f.gen.Beer().Name() }
func (f FakerBeer) Style() string { return f.gen.Beer().Style() }
func (f FakerBeer) Hop() string   { return f.gen.Beer().Hop() }
func (f FakerBeer) Malt() string  { return f.gen.Beer().Malt() }

type FakerMusic struct{ gen faker.Faker }

func (f FakerMusic) Name() string   { return f.gen.Music().Name() }
func (f FakerMusic) Genre() string  { return f.gen.Music().Genre() }
func (f FakerMusic) Author() string { return f.gen.Music().Author() }

type FakerPet struct{ gen faker.Faker }

func (f FakerPet) Name() string { return f.gen.Pet().Name() }
func (f FakerPet) Cat() string  { return f.gen.Pet().Cat() }
func (f FakerPet) Dog() string  { return f.gen.Pet().Dog() }

type FakerApp struct{ gen faker.Faker }

func (f FakerApp) Name() string    { return f.gen.App().Name() }
func (f FakerApp) Version() string { return f.gen.App().Version() }

type FakerBlood struct{ gen faker.Faker }

func (f FakerBlood) Name() string { return f.gen.Blood().Name() }

type FakerBoolean struct{ gen faker.Faker }

func (f FakerBoolean) Bool() bool { return f.gen.Boolean().Bool() }

type FakerCrypto struct{ gen faker.Faker }

func (f FakerCrypto) BitcoinAddress() string   { return f.gen.Crypto().BitcoinAddress() }
func (f FakerCrypto) EtheriumAddress() string   { return f.gen.Crypto().EtheriumAddress() }

type FakerEmoji struct{ gen faker.Faker }

func (f FakerEmoji) Emoji() string     { return f.gen.Emoji().Emoji() }
func (f FakerEmoji) EmojiCode() string { return f.gen.Emoji().EmojiCode() }

type FakerFile struct{ gen faker.Faker }

func (f FakerFile) Extension() string              { return f.gen.File().Extension() }
func (f FakerFile) FilenameWithExtension() string   { return f.gen.File().FilenameWithExtension() }

type FakerGamer struct{ gen faker.Faker }

func (f FakerGamer) Tag() string { return f.gen.Gamer().Tag() }

type FakerGender struct{ gen faker.Faker }

func (f FakerGender) Name() string { return f.gen.Gender().Name() }
func (f FakerGender) Abbr() string { return f.gen.Gender().Abbr() }

type FakerGenre struct{ gen faker.Faker }

func (f FakerGenre) Name() string { return f.gen.Genre().Name() }

type FakerLanguage struct{ gen faker.Faker }

func (f FakerLanguage) Language() string            { return f.gen.Language().Language() }
func (f FakerLanguage) LanguageAbbr() string        { return f.gen.Language().LanguageAbbr() }
func (f FakerLanguage) ProgrammingLanguage() string { return f.gen.Language().ProgrammingLanguage() }

type FakerMimeType struct{ gen faker.Faker }

func (f FakerMimeType) MimeType() string { return f.gen.MimeType().MimeType() }

type FakerPokemon struct{ gen faker.Faker }

func (f FakerPokemon) English() string  { return f.gen.Pokemon().English() }
func (f FakerPokemon) Japanese() string { return f.gen.Pokemon().Japanese() }

type FakerYouTube struct{ gen faker.Faker }

func (f FakerYouTube) GenerateVideoID() string  { return f.gen.YouTube().GenerateVideoID() }
func (f FakerYouTube) GenerateFullURL() string   { return f.gen.YouTube().GenerateFullURL() }
func (f FakerYouTube) GenerateShareURL() string  { return f.gen.YouTube().GenerateShareURL() }
func (f FakerYouTube) GenerateEmbededURL() string { return f.gen.YouTube().GenerateEmbededURL() }
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go test ./internal/infrastructure/outbound/faker/... -v`

Expected: All tests PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/outbound/faker/
git commit -m "feat: add faker wrapper package with all category structs"
```

---

### Task 3: Add FakerSeed to YAML, domain, and compiled types

**Files:**
- Modify: `internal/infrastructure/outbound/filesystem/yaml_types.go:33-40`
- Modify: `internal/domain/scenario/scenario.go:66-73`
- Modify: `internal/domain/match/predicate.go:70-79`
- Modify: `internal/infrastructure/outbound/filesystem/repository.go:373-381`

**Step 1: Add `FakerSeed` to `yamlResponse`**

In `internal/infrastructure/outbound/filesystem/yaml_types.go`, add to `yamlResponse`:

```go
type yamlResponse struct {
	Status      int               `yaml:"status"`
	Headers     map[string]string `yaml:"headers,omitempty"`
	Body        string            `yaml:"body,omitempty"`
	BodyFile    string            `yaml:"body_file,omitempty"`
	ContentType string            `yaml:"content_type,omitempty"`
	Engine      string            `yaml:"engine,omitempty"`
	FakerSeed   *int64            `yaml:"faker_seed,omitempty"`
}
```

**Step 2: Add `FakerSeed` to domain `Response`**

In `internal/domain/scenario/scenario.go`, add to `Response`:

```go
type Response struct {
	Status      int
	Headers     map[string]string
	Body        string
	BodyFile    string
	ContentType string
	Engine      string
	FakerSeed   *int64
}
```

**Step 3: Add `FakerSeed` to `RenderContext`**

In `internal/domain/match/predicate.go`, add to `RenderContext`:

```go
type RenderContext struct {
	Method      string
	Path        string
	Headers     map[string]string
	QueryParams map[string]string
	PathParams  map[string]string
	Body        []byte
	Now         string
	FakerSeed   *int64
}
```

**Step 4: Wire FakerSeed in repository `toScenario()`**

In `internal/infrastructure/outbound/filesystem/repository.go`, update the `toScenario()` function's Response block:

```go
Response: scenario.Response{
	Status:      ys.Response.Status,
	Headers:     ys.Response.Headers,
	Body:        ys.Response.Body,
	BodyFile:    ys.Response.BodyFile,
	ContentType: ys.Response.ContentType,
	Engine:      ys.Response.Engine,
	FakerSeed:   ys.Response.FakerSeed,
},
```

**Step 5: Add `FakerSeed` to `CompiledResponse`**

In `internal/domain/match/predicate.go`, add to `CompiledResponse`:

```go
type CompiledResponse struct {
	Status      int
	Headers     map[string]string
	Body        []byte
	Renderer    BodyRenderer
	ContentType string
	FakerSeed   *int64
}
```

**Step 6: Wire FakerSeed in compiler `compileResponse()`**

In `internal/infrastructure/services/compiler.go`, update `compileResponse()` to include:

```go
resp := match.CompiledResponse{
	Status:      r.Status,
	Headers:     r.Headers,
	ContentType: r.ContentType,
	FakerSeed:   r.FakerSeed,
}
```

**Step 7: Verify everything compiles**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go build ./...`

Expected: BUILD SUCCESS

**Step 8: Commit**

```bash
git add internal/infrastructure/outbound/filesystem/yaml_types.go \
        internal/domain/scenario/scenario.go \
        internal/domain/match/predicate.go \
        internal/infrastructure/outbound/filesystem/repository.go \
        internal/infrastructure/services/compiler.go
git commit -m "feat: add FakerSeed field through YAML, domain, and compiled types"
```

---

### Task 4: Integrate faker into Expr template engine

**Files:**
- Modify: `internal/infrastructure/outbound/template/expr.go:113-126`
- Modify: `internal/infrastructure/outbound/template/functions.go:1-64`
- Test: `internal/infrastructure/outbound/template/expr_test.go`

**Step 1: Write the failing test**

Add to `internal/infrastructure/outbound/template/expr_test.go`:

```go
func TestExprCompiler_FakerPerson(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `${faker.Person.Name()}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if len(string(result)) == 0 {
		t.Error("expected non-empty faker person name")
	}
}

func TestExprCompiler_FakerSeeded(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `${faker.Person.Name()}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	seed := int64(42)
	r1, _ := renderer.Render(match.RenderContext{FakerSeed: &seed})
	r2, _ := renderer.Render(match.RenderContext{FakerSeed: &seed})

	if string(r1) != string(r2) {
		t.Errorf("seeded faker should be deterministic: %q != %q", r1, r2)
	}
}

func TestExprCompiler_FakerMultipleCategories(t *testing.T) {
	c := &ExprCompiler{}
	renderer, err := c.Compile("test", `${faker.Person.Name()} from ${faker.Address.City()}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	s := string(result)
	if !strings.Contains(s, " from ") {
		t.Errorf("expected 'X from Y' format, got %q", s)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go test ./internal/infrastructure/outbound/template/ -run TestExprCompiler_Faker -v`

Expected: FAIL — `faker` not recognized in exprEnv

**Step 3: Add faker to exprEnv and buildExprEnv**

In `internal/infrastructure/outbound/template/expr.go`, add the `Faker` field to `exprEnv`:

```go
type exprEnv struct {
	PathParam  func(string) string  `expr:"pathParam"`
	QueryParam func(string) string  `expr:"queryParam"`
	Header     func(string) string  `expr:"header"`
	Body       func() string        `expr:"body"`
	Now        func() string        `expr:"now"`
	NowFormat  func(string) string  `expr:"nowFormat"`
	UUID       func() string        `expr:"uuid"`
	RandomInt  func(int, int) int   `expr:"randomInt"`
	Seq        func(int, int) []int `expr:"seq"`
	ToJSON     func(any) string     `expr:"toJSON"`
	JsonPath   func(string) string  `expr:"jsonPath"`
	Faker      fakerpkg.FakerContext `expr:"faker"`
}
```

Add the import at the top of `expr.go`:

```go
import (
	// ... existing imports
	fakerpkg "github.com/sophialabs/proteusmock/internal/infrastructure/outbound/faker"
)
```

In `internal/infrastructure/outbound/template/functions.go`, add faker creation to `buildExprEnv()`:

After the closing brace of the return statement (before the final `}`), add the Faker field:

```go
func buildExprEnv(ctx match.RenderContext) exprEnv {
	return exprEnv{
		// ... all existing fields unchanged ...
		Faker: fakerpkg.NewFakerContext(ctx.FakerSeed),
	}
}
```

**Step 4: Run tests**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go test ./internal/infrastructure/outbound/template/ -v`

Expected: All tests PASS (both new faker tests and existing tests)

**Step 5: Commit**

```bash
git add internal/infrastructure/outbound/template/expr.go \
        internal/infrastructure/outbound/template/functions.go \
        internal/infrastructure/outbound/template/expr_test.go
git commit -m "feat: integrate faker into Expr template engine"
```

---

### Task 5: Integrate faker into Jinja2 template engine

**Files:**
- Modify: `internal/infrastructure/outbound/template/jinja2.go:28-65`
- Test: `internal/infrastructure/outbound/template/jinja2_test.go`

**Step 1: Write the failing test**

Add to `internal/infrastructure/outbound/template/jinja2_test.go`:

```go
func TestJinja2Compiler_FakerPerson(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ faker.Person.Name }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if len(string(result)) == 0 {
		t.Error("expected non-empty faker person name")
	}
}

func TestJinja2Compiler_FakerSeeded(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ faker.Person.Name }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	seed := int64(42)
	r1, _ := renderer.Render(match.RenderContext{FakerSeed: &seed})
	r2, _ := renderer.Render(match.RenderContext{FakerSeed: &seed})

	if string(r1) != string(r2) {
		t.Errorf("seeded faker should be deterministic: %q != %q", r1, r2)
	}
}

func TestJinja2Compiler_FakerMultipleCategories(t *testing.T) {
	c := &Jinja2Compiler{}
	renderer, err := c.Compile("test", `{{ faker.Internet.Email }}`)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	result, err := renderer.Render(match.RenderContext{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	s := string(result)
	if !strings.Contains(s, "@") {
		t.Errorf("expected email with @, got %q", s)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go test ./internal/infrastructure/outbound/template/ -run TestJinja2Compiler_Faker -v`

Expected: FAIL — `faker` not in Pongo2 context

**Step 3: Add faker to Jinja2 render context**

In `internal/infrastructure/outbound/template/jinja2.go`, add the import and modify `Render()`:

Add import:
```go
import (
	// ... existing imports
	fakerpkg "github.com/sophialabs/proteusmock/internal/infrastructure/outbound/faker"
)
```

In the `Render` method, add `faker` to `pongoCtx`:

```go
func (r *jinja2Renderer) Render(ctx match.RenderContext) ([]byte, error) {
	pongoCtx := pongo2.Context{
		// ... all existing entries unchanged ...
		"faker": fakerpkg.NewFakerContext(ctx.FakerSeed),
	}
	// ... rest unchanged
}
```

**Step 4: Run tests**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go test ./internal/infrastructure/outbound/template/ -v`

Expected: All tests PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/outbound/template/jinja2.go \
        internal/infrastructure/outbound/template/jinja2_test.go
git commit -m "feat: integrate faker into Jinja2 template engine"
```

---

### Task 6: Wire FakerSeed through server render context

**Files:**
- Modify: `internal/infrastructure/inbound/http/server.go:271-279`

**Step 1: Update RenderContext construction in server.go**

In `internal/infrastructure/inbound/http/server.go`, around line 271, update the `renderCtx` construction:

```go
renderCtx := match.RenderContext{
	Method:      r.Method,
	Path:        r.URL.Path,
	Headers:     headers,
	QueryParams: queryParams,
	PathParams:  extractPathParams(r),
	Body:        body,
	Now:         time.Now().UTC().Format(time.RFC3339),
	FakerSeed:   resp.FakerSeed,
}
```

**Step 2: Verify build**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go build ./...`

Expected: BUILD SUCCESS

**Step 3: Run all tests**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go test ./... -v`

Expected: All tests PASS

**Step 4: Commit**

```bash
git add internal/infrastructure/inbound/http/server.go
git commit -m "feat: wire FakerSeed through server render context"
```

---

### Task 7: Add example scenario

**Files:**
- Create: `mock/scenarios/showcase/faker.yaml`

**Step 1: Create showcase scenario**

Create `mock/scenarios/showcase/faker.yaml`:

```yaml
- id: showcase-faker-random
  name: "Faker: random user data"
  priority: 1
  when:
    method: GET
    path: /api/v1/fake/user
  response:
    status: 200
    engine: expr
    headers:
      Content-Type: application/json
    body: |
      {
        "name": "${faker.Person.Name()}",
        "email": "${faker.Internet.Email()}",
        "phone": "${faker.Phone.Number()}",
        "address": {
          "street": "${faker.Address.StreetAddress()}",
          "city": "${faker.Address.City()}",
          "state": "${faker.Address.State()}",
          "zip": "${faker.Address.PostCode()}",
          "country": "${faker.Address.Country()}"
        },
        "company": "${faker.Company.Name()}",
        "job_title": "${faker.Company.JobTitle()}"
      }

- id: showcase-faker-seeded
  name: "Faker: deterministic user data"
  priority: 1
  when:
    method: GET
    path: /api/v1/fake/user/deterministic
  response:
    status: 200
    engine: expr
    faker_seed: 42
    headers:
      Content-Type: application/json
    body: |
      {
        "name": "${faker.Person.Name()}",
        "email": "${faker.Internet.Email()}",
        "ipv4": "${faker.Internet.Ipv4()}"
      }

- id: showcase-faker-jinja2
  name: "Faker: Jinja2 user list"
  priority: 1
  when:
    method: GET
    path: /api/v1/fake/products
  response:
    status: 200
    engine: jinja2
    headers:
      Content-Type: application/json
    body: |
      [
        {% for i in seq(1, 3) %}
        {
          "id": {{ i }},
          "name": "{{ faker.App.Name }}",
          "color": "{{ faker.Color.ColorName }}",
          "price": "{{ faker.Currency.Code }}"
        }{% if not forloop.Last %},{% endif %}
        {% endfor %}
      ]
```

**Step 2: Commit**

```bash
git add mock/scenarios/showcase/faker.yaml
git commit -m "feat: add faker showcase scenario examples"
```

---

### Task 8: Run full test suite and verify

**Step 1: Run all tests**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go test ./... -v -count=1`

Expected: All tests PASS

**Step 2: Build and run smoke test**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go build -o /tmp/proteusmock ./cmd/proteusmock`

Expected: BUILD SUCCESS

**Step 3: Verify with `go vet`**

Run: `cd /Users/nadilsonferreira/Developer/sophia/proteusmock && go vet ./...`

Expected: No issues

---

### Important Notes

- **Expr engine syntax**: Expr uses Go method call syntax — `faker.Person.Name()` with parentheses and PascalCase field/method names. The `expr:"faker"` struct tag maps the Go field to the lowercase `faker` name in expressions.
- **Jinja2/Pongo2 syntax**: Pongo2 accesses struct methods as properties — `faker.Person.Name` (no parentheses for zero-arg methods). It resolves PascalCase struct fields via Go reflection.
- **Seeding**: Each render call creates a fresh `FakerContext`. With the same seed, the sequence is always the same. Without a seed, `faker.New()` uses `math/rand` global source.
- **No Jinja2 parentheses**: In Pongo2, method calls on context objects look like property access: `{{ faker.Person.Name }}` not `{{ faker.Person.Name() }}`. This is a Pongo2 behavior — it auto-calls zero-argument methods.
