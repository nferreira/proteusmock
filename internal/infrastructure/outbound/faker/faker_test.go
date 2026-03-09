package faker

import (
	"testing"
)

func TestNewFakerContext_Unseeded(t *testing.T) {
	ctx := NewFakerContext(nil)

	if name := ctx.Person.Name(); name == "" {
		t.Error("Person.Name() returned empty string")
	}
	if addr := ctx.Address.City(); addr == "" {
		t.Error("Address.City() returned empty string")
	}
	if email := ctx.Internet.Email(); email == "" {
		t.Error("Internet.Email() returned empty string")
	}
	if company := ctx.Company.Name(); company == "" {
		t.Error("Company.Name() returned empty string")
	}
}

func TestNewFakerContext_Seeded_Deterministic(t *testing.T) {
	seed := int64(42)

	ctx1 := NewFakerContext(&seed)
	ctx2 := NewFakerContext(&seed)

	name1 := ctx1.Person.Name()
	name2 := ctx2.Person.Name()
	if name1 != name2 {
		t.Errorf("seeded Person.Name() not deterministic: %q != %q", name1, name2)
	}

	ctx3 := NewFakerContext(&seed)
	ctx4 := NewFakerContext(&seed)
	email1 := ctx3.Internet.Email()
	email2 := ctx4.Internet.Email()
	if email1 != email2 {
		t.Errorf("seeded Internet.Email() not deterministic: %q != %q", email1, email2)
	}

	ctx5 := NewFakerContext(&seed)
	ctx6 := NewFakerContext(&seed)
	city1 := ctx5.Address.City()
	city2 := ctx6.Address.City()
	if city1 != city2 {
		t.Errorf("seeded Address.City() not deterministic: %q != %q", city1, city2)
	}
}

func TestNewFakerContext_AllCategories(t *testing.T) {
	ctx := NewFakerContext(nil)

	tests := []struct {
		category string
		fn       func() string
	}{
		// Person
		{"Person.Name", func() string { return ctx.Person.Name() }},
		{"Person.FirstName", func() string { return ctx.Person.FirstName() }},
		{"Person.LastName", func() string { return ctx.Person.LastName() }},
		// Address
		{"Address.City", func() string { return ctx.Address.City() }},
		{"Address.Country", func() string { return ctx.Address.Country() }},
		{"Address.StreetName", func() string { return ctx.Address.StreetName() }},
		// Internet
		{"Internet.Email", func() string { return ctx.Internet.Email() }},
		{"Internet.URL", func() string { return ctx.Internet.URL() }},
		{"Internet.Ipv4", func() string { return ctx.Internet.Ipv4() }},
		// Company
		{"Company.Name", func() string { return ctx.Company.Name() }},
		{"Company.JobTitle", func() string { return ctx.Company.JobTitle() }},
		// Lorem
		{"Lorem.Word", func() string { return ctx.Lorem.Word() }},
		{"Lorem.Sentence", func() string { return ctx.Lorem.Sentence() }},
		{"Lorem.Paragraph", func() string { return ctx.Lorem.Paragraph() }},
		// Payment
		{"Payment.CreditCardNumber", func() string { return ctx.Payment.CreditCardNumber() }},
		{"Payment.CreditCardType", func() string { return ctx.Payment.CreditCardType() }},
		{"Payment.Iban", func() string { return ctx.Payment.Iban() }},
		// Currency
		{"Currency.Currency", func() string { return ctx.Currency.Currency() }},
		{"Currency.Code", func() string { return ctx.Currency.Code() }},
		{"Currency.Country", func() string { return ctx.Currency.Country() }},
		// Phone
		{"Phone.Number", func() string { return ctx.Phone.Number() }},
		{"Phone.E164Number", func() string { return ctx.Phone.E164Number() }},
		// Hash
		{"Hash.MD5", func() string { return ctx.Hash.MD5() }},
		{"Hash.SHA256", func() string { return ctx.Hash.SHA256() }},
		{"Hash.SHA512", func() string { return ctx.Hash.SHA512() }},
		// UUID
		{"UUID.V4", func() string { return ctx.UUID.V4() }},
		// UserAgent
		{"UserAgent.UserAgent", func() string { return ctx.UserAgent.UserAgent() }},
		{"UserAgent.Chrome", func() string { return ctx.UserAgent.Chrome() }},
		// Color
		{"Color.Hex", func() string { return ctx.Color.Hex() }},
		{"Color.ColorName", func() string { return ctx.Color.ColorName() }},
		{"Color.CSS", func() string { return ctx.Color.CSS() }},
		// Time
		{"Time.DayOfWeek", func() string { return ctx.Time.DayOfWeek() }},
		{"Time.MonthName", func() string { return ctx.Time.MonthName() }},
		{"Time.Century", func() string { return ctx.Time.Century() }},
		{"Time.Timezone", func() string { return ctx.Time.Timezone() }},
		// Car
		{"Car.Maker", func() string { return ctx.Car.Maker() }},
		{"Car.Model", func() string { return ctx.Car.Model() }},
		{"Car.Plate", func() string { return ctx.Car.Plate() }},
		// Food
		{"Food.Fruit", func() string { return ctx.Food.Fruit() }},
		{"Food.Vegetable", func() string { return ctx.Food.Vegetable() }},
		// Beer
		{"Beer.Name", func() string { return ctx.Beer.Name() }},
		{"Beer.Style", func() string { return ctx.Beer.Style() }},
		// Music
		{"Music.Name", func() string { return ctx.Music.Name() }},
		{"Music.Genre", func() string { return ctx.Music.Genre() }},
		{"Music.Author", func() string { return ctx.Music.Author() }},
		// Pet
		{"Pet.Name", func() string { return ctx.Pet.Name() }},
		{"Pet.Cat", func() string { return ctx.Pet.Cat() }},
		{"Pet.Dog", func() string { return ctx.Pet.Dog() }},
		// App
		{"App.Name", func() string { return ctx.App.Name() }},
		{"App.Version", func() string { return ctx.App.Version() }},
		// Blood
		{"Blood.Name", func() string { return ctx.Blood.Name() }},
		// Crypto
		{"Crypto.BitcoinAddress", func() string { return ctx.Crypto.BitcoinAddress() }},
		{"Crypto.EtheriumAddress", func() string { return ctx.Crypto.EtheriumAddress() }},
		// Emoji
		{"Emoji.Emoji", func() string { return ctx.Emoji.Emoji() }},
		{"Emoji.EmojiCode", func() string { return ctx.Emoji.EmojiCode() }},
		// File
		{"File.Extension", func() string { return ctx.File.Extension() }},
		{"File.FilenameWithExtension", func() string { return ctx.File.FilenameWithExtension() }},
		// Gamer
		{"Gamer.Tag", func() string { return ctx.Gamer.Tag() }},
		// Gender
		{"Gender.Name", func() string { return ctx.Gender.Name() }},
		{"Gender.Abbr", func() string { return ctx.Gender.Abbr() }},
		// Genre
		{"Genre.Name", func() string { return ctx.Genre.Name() }},
		// Language
		{"Language.Language", func() string { return ctx.Language.Language() }},
		{"Language.LanguageAbbr", func() string { return ctx.Language.LanguageAbbr() }},
		{"Language.ProgrammingLanguage", func() string { return ctx.Language.ProgrammingLanguage() }},
		// MimeType
		{"MimeType.MimeType", func() string { return ctx.MimeType.MimeType() }},
		// Pokemon
		{"Pokemon.English", func() string { return ctx.Pokemon.English() }},
		{"Pokemon.Japanese", func() string { return ctx.Pokemon.Japanese() }},
		// YouTube
		{"YouTube.GenerateVideoID", func() string { return ctx.YouTube.GenerateVideoID() }},
		{"YouTube.GenerateFullURL", func() string { return ctx.YouTube.GenerateFullURL() }},
		{"YouTube.GenerateShareURL", func() string { return ctx.YouTube.GenerateShareURL() }},
		{"YouTube.GenerateEmbededURL", func() string { return ctx.YouTube.GenerateEmbededURL() }},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			result := tt.fn()
			if result == "" {
				t.Errorf("%s returned empty string", tt.category)
			}
		})
	}
}
