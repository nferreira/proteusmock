// Package faker provides a thin wrapper around github.com/jaswdr/faker/v2,
// exposing every faker category as a struct with zero-arg methods suitable
// for use inside template expressions.
package faker

import (
	"fmt"
	"math/rand/v2"

	"github.com/jaswdr/faker/v2"
)

// FakerContext holds wrapper structs for every faker category.
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

// NewFakerContext creates a FakerContext. When seed is non-nil the
// underlying generator is deterministic; otherwise it is random.
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

// ---------------------------------------------------------------------------
// Person
// ---------------------------------------------------------------------------

type FakerPerson struct{ gen faker.Faker }

func (f FakerPerson) Name() string            { return f.gen.Person().Name() }
func (f FakerPerson) FirstName() string        { return f.gen.Person().FirstName() }
func (f FakerPerson) FirstNameMale() string    { return f.gen.Person().FirstNameMale() }
func (f FakerPerson) FirstNameFemale() string  { return f.gen.Person().FirstNameFemale() }
func (f FakerPerson) LastName() string         { return f.gen.Person().LastName() }
func (f FakerPerson) Title() string            { return f.gen.Person().Title() }
func (f FakerPerson) TitleMale() string        { return f.gen.Person().TitleMale() }
func (f FakerPerson) TitleFemale() string      { return f.gen.Person().TitleFemale() }
func (f FakerPerson) Suffix() string           { return f.gen.Person().Suffix() }
func (f FakerPerson) Gender() string           { return f.gen.Person().Gender() }
func (f FakerPerson) GenderMale() string       { return f.gen.Person().GenderMale() }
func (f FakerPerson) GenderFemale() string     { return f.gen.Person().GenderFemale() }
func (f FakerPerson) NameMale() string         { return f.gen.Person().NameMale() }
func (f FakerPerson) NameFemale() string       { return f.gen.Person().NameFemale() }
func (f FakerPerson) SSN() string              { return f.gen.Person().SSN() }

// ---------------------------------------------------------------------------
// Address
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Internet
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Company
// ---------------------------------------------------------------------------

type FakerCompany struct{ gen faker.Faker }

func (f FakerCompany) Name() string       { return f.gen.Company().Name() }
func (f FakerCompany) Suffix() string     { return f.gen.Company().Suffix() }
func (f FakerCompany) CatchPhrase() string { return f.gen.Company().CatchPhrase() }
func (f FakerCompany) BS() string         { return f.gen.Company().BS() }
func (f FakerCompany) JobTitle() string   { return f.gen.Company().JobTitle() }
func (f FakerCompany) EIN() string        { return fmt.Sprintf("%d", f.gen.Company().EIN()) }

// ---------------------------------------------------------------------------
// Lorem
// ---------------------------------------------------------------------------

type FakerLorem struct{ gen faker.Faker }

func (f FakerLorem) Word() string      { return f.gen.Lorem().Word() }
func (f FakerLorem) Sentence() string  { return f.gen.Lorem().Sentence(6) }
func (f FakerLorem) Paragraph() string { return f.gen.Lorem().Paragraph(3) }

// ---------------------------------------------------------------------------
// Payment
// ---------------------------------------------------------------------------

type FakerPayment struct{ gen faker.Faker }

func (f FakerPayment) CreditCardNumber() string               { return f.gen.Payment().CreditCardNumber() }
func (f FakerPayment) CreditCardType() string                 { return f.gen.Payment().CreditCardType() }
func (f FakerPayment) CreditCardExpirationDateString() string { return f.gen.Payment().CreditCardExpirationDateString() }
func (f FakerPayment) Iban() string                           { return f.gen.Payment().Iban() }

// ---------------------------------------------------------------------------
// Currency
// ---------------------------------------------------------------------------

type FakerCurrency struct{ gen faker.Faker }

func (f FakerCurrency) Currency() string { return f.gen.Currency().Currency() }
func (f FakerCurrency) Code() string     { return f.gen.Currency().Code() }
func (f FakerCurrency) Number() string   { return fmt.Sprintf("%d", f.gen.Currency().Number()) }
func (f FakerCurrency) Country() string  { return f.gen.Currency().Country() }

// ---------------------------------------------------------------------------
// Phone
// ---------------------------------------------------------------------------

type FakerPhone struct{ gen faker.Faker }

func (f FakerPhone) Number() string           { return f.gen.Phone().Number() }
func (f FakerPhone) E164Number() string       { return f.gen.Phone().E164Number() }
func (f FakerPhone) AreaCode() string         { return f.gen.Phone().AreaCode() }
func (f FakerPhone) ExchangeCode() string     { return f.gen.Phone().ExchangeCode() }
func (f FakerPhone) TollFreeAreaCode() string { return f.gen.Phone().TollFreeAreaCode() }
func (f FakerPhone) ToolFreeNumber() string   { return f.gen.Phone().ToolFreeNumber() }

// ---------------------------------------------------------------------------
// Hash
// ---------------------------------------------------------------------------

type FakerHash struct{ gen faker.Faker }

func (f FakerHash) MD5() string    { return f.gen.Hash().MD5() }
func (f FakerHash) SHA256() string { return f.gen.Hash().SHA256() }
func (f FakerHash) SHA512() string { return f.gen.Hash().SHA512() }

// ---------------------------------------------------------------------------
// UUID
// ---------------------------------------------------------------------------

type FakerUUID struct{ gen faker.Faker }

func (f FakerUUID) V4() string { return f.gen.UUID().V4() }

// ---------------------------------------------------------------------------
// UserAgent
// ---------------------------------------------------------------------------

type FakerUserAgent struct{ gen faker.Faker }

func (f FakerUserAgent) UserAgent() string         { return f.gen.UserAgent().UserAgent() }
func (f FakerUserAgent) Chrome() string            { return f.gen.UserAgent().Chrome() }
func (f FakerUserAgent) Firefox() string           { return f.gen.UserAgent().Firefox() }
func (f FakerUserAgent) Safari() string            { return f.gen.UserAgent().Safari() }
func (f FakerUserAgent) InternetExplorer() string  { return f.gen.UserAgent().InternetExplorer() }
func (f FakerUserAgent) Opera() string             { return f.gen.UserAgent().Opera() }

// ---------------------------------------------------------------------------
// Color
// ---------------------------------------------------------------------------

type FakerColor struct{ gen faker.Faker }

func (f FakerColor) Hex() string           { return f.gen.Color().Hex() }
func (f FakerColor) RGB() string           { return f.gen.Color().RGB() }
func (f FakerColor) ColorName() string     { return f.gen.Color().ColorName() }
func (f FakerColor) SafeColorName() string { return f.gen.Color().SafeColorName() }
func (f FakerColor) CSS() string           { return f.gen.Color().CSS() }

// ---------------------------------------------------------------------------
// Time
// ---------------------------------------------------------------------------

type FakerTime struct{ gen faker.Faker }

func (f FakerTime) DayOfWeek() string  { return f.gen.Time().DayOfWeek().String() }
func (f FakerTime) DayOfMonth() string { return fmt.Sprintf("%d", f.gen.Time().DayOfMonth()) }
func (f FakerTime) Month() string      { return f.gen.Time().Month().String() }
func (f FakerTime) MonthName() string  { return f.gen.Time().MonthName() }
func (f FakerTime) Year() string       { return fmt.Sprintf("%d", f.gen.Time().Year()) }
func (f FakerTime) Century() string    { return f.gen.Time().Century() }
func (f FakerTime) Timezone() string   { return f.gen.Time().Timezone() }
func (f FakerTime) AmPm() string       { return f.gen.Time().AmPm() }

// ---------------------------------------------------------------------------
// Car
// ---------------------------------------------------------------------------

type FakerCar struct{ gen faker.Faker }

func (f FakerCar) Maker() string            { return f.gen.Car().Maker() }
func (f FakerCar) Model() string            { return f.gen.Car().Model() }
func (f FakerCar) Category() string         { return f.gen.Car().Category() }
func (f FakerCar) FuelType() string         { return f.gen.Car().FuelType() }
func (f FakerCar) TransmissionGear() string { return f.gen.Car().TransmissionGear() }
func (f FakerCar) Plate() string            { return f.gen.Car().Plate() }

// ---------------------------------------------------------------------------
// Food
// ---------------------------------------------------------------------------

type FakerFood struct{ gen faker.Faker }

func (f FakerFood) Fruit() string     { return f.gen.Food().Fruit() }
func (f FakerFood) Vegetable() string { return f.gen.Food().Vegetable() }

// ---------------------------------------------------------------------------
// Beer
// ---------------------------------------------------------------------------

type FakerBeer struct{ gen faker.Faker }

func (f FakerBeer) Name() string  { return f.gen.Beer().Name() }
func (f FakerBeer) Style() string { return f.gen.Beer().Style() }
func (f FakerBeer) Hop() string   { return f.gen.Beer().Hop() }
func (f FakerBeer) Malt() string  { return f.gen.Beer().Malt() }

// ---------------------------------------------------------------------------
// Music
// ---------------------------------------------------------------------------

type FakerMusic struct{ gen faker.Faker }

func (f FakerMusic) Name() string   { return f.gen.Music().Name() }
func (f FakerMusic) Genre() string  { return f.gen.Music().Genre() }
func (f FakerMusic) Author() string { return f.gen.Music().Author() }

// ---------------------------------------------------------------------------
// Pet
// ---------------------------------------------------------------------------

type FakerPet struct{ gen faker.Faker }

func (f FakerPet) Name() string { return f.gen.Pet().Name() }
func (f FakerPet) Cat() string  { return f.gen.Pet().Cat() }
func (f FakerPet) Dog() string  { return f.gen.Pet().Dog() }

// ---------------------------------------------------------------------------
// App
// ---------------------------------------------------------------------------

type FakerApp struct{ gen faker.Faker }

func (f FakerApp) Name() string    { return f.gen.App().Name() }
func (f FakerApp) Version() string { return f.gen.App().Version() }

// ---------------------------------------------------------------------------
// Blood
// ---------------------------------------------------------------------------

type FakerBlood struct{ gen faker.Faker }

func (f FakerBlood) Name() string { return f.gen.Blood().Name() }

// ---------------------------------------------------------------------------
// Boolean
// ---------------------------------------------------------------------------

type FakerBoolean struct{ gen faker.Faker }

func (f FakerBoolean) Bool() bool { return f.gen.Boolean().Bool() }

// ---------------------------------------------------------------------------
// Crypto
// ---------------------------------------------------------------------------

type FakerCrypto struct{ gen faker.Faker }

func (f FakerCrypto) BitcoinAddress() string  { return f.gen.Crypto().BitcoinAddress() }
func (f FakerCrypto) EtheriumAddress() string { return f.gen.Crypto().EtheriumAddress() }

// ---------------------------------------------------------------------------
// Emoji
// ---------------------------------------------------------------------------

type FakerEmoji struct{ gen faker.Faker }

func (f FakerEmoji) Emoji() string     { return f.gen.Emoji().Emoji() }
func (f FakerEmoji) EmojiCode() string { return f.gen.Emoji().EmojiCode() }

// ---------------------------------------------------------------------------
// File
// ---------------------------------------------------------------------------

type FakerFile struct{ gen faker.Faker }

func (f FakerFile) Extension() string             { return f.gen.File().Extension() }
func (f FakerFile) FilenameWithExtension() string  { return f.gen.File().FilenameWithExtension() }

// ---------------------------------------------------------------------------
// Gamer
// ---------------------------------------------------------------------------

type FakerGamer struct{ gen faker.Faker }

func (f FakerGamer) Tag() string { return f.gen.Gamer().Tag() }

// ---------------------------------------------------------------------------
// Gender
// ---------------------------------------------------------------------------

type FakerGender struct{ gen faker.Faker }

func (f FakerGender) Name() string { return f.gen.Gender().Name() }
func (f FakerGender) Abbr() string { return f.gen.Gender().Abbr() }

// ---------------------------------------------------------------------------
// Genre
// ---------------------------------------------------------------------------

type FakerGenre struct{ gen faker.Faker }

func (f FakerGenre) Name() string { return f.gen.Genre().Name() }

// ---------------------------------------------------------------------------
// Language
// ---------------------------------------------------------------------------

type FakerLanguage struct{ gen faker.Faker }

func (f FakerLanguage) Language() string              { return f.gen.Language().Language() }
func (f FakerLanguage) LanguageAbbr() string          { return f.gen.Language().LanguageAbbr() }
func (f FakerLanguage) ProgrammingLanguage() string   { return f.gen.Language().ProgrammingLanguage() }

// ---------------------------------------------------------------------------
// MimeType
// ---------------------------------------------------------------------------

type FakerMimeType struct{ gen faker.Faker }

func (f FakerMimeType) MimeType() string { return f.gen.MimeType().MimeType() }

// ---------------------------------------------------------------------------
// Pokemon
// ---------------------------------------------------------------------------

type FakerPokemon struct{ gen faker.Faker }

func (f FakerPokemon) English() string  { return f.gen.Pokemon().English() }
func (f FakerPokemon) Japanese() string { return f.gen.Pokemon().Japanese() }

// ---------------------------------------------------------------------------
// YouTube
// ---------------------------------------------------------------------------

type FakerYouTube struct{ gen faker.Faker }

func (f FakerYouTube) GenerateVideoID() string   { return f.gen.YouTube().GenerateVideoID() }
func (f FakerYouTube) GenerateFullURL() string    { return f.gen.YouTube().GenerateFullURL() }
func (f FakerYouTube) GenerateShareURL() string   { return f.gen.YouTube().GenerateShareURL() }
func (f FakerYouTube) GenerateEmbededURL() string { return f.gen.YouTube().GenerateEmbededURL() }
