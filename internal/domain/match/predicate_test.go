package match_test

import (
	"testing"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

func TestAnd(t *testing.T) {
	p := match.And(
		func(s string) bool { return len(s) > 2 },
		func(s string) bool { return s[0] == 'a' },
	)

	if !p("abc") {
		t.Error("expected match for 'abc'")
	}
	if p("ab") {
		t.Error("expected no match for 'ab'")
	}
	if p("xyz") {
		t.Error("expected no match for 'xyz'")
	}
}

func TestOr(t *testing.T) {
	p := match.Or(
		func(s string) bool { return s == "hello" },
		func(s string) bool { return s == "world" },
	)

	if !p("hello") {
		t.Error("expected match for 'hello'")
	}
	if !p("world") {
		t.Error("expected match for 'world'")
	}
	if p("other") {
		t.Error("expected no match for 'other'")
	}
}

func TestNot(t *testing.T) {
	p := match.Not(func(s string) bool { return s == "no" })

	if !p("yes") {
		t.Error("expected match for 'yes'")
	}
	if p("no") {
		t.Error("expected no match for 'no'")
	}
}

func TestAlways(t *testing.T) {
	p := match.Always()
	if !p("anything") {
		t.Error("Always should match everything")
	}
}

func TestNever(t *testing.T) {
	p := match.Never()
	if p("anything") {
		t.Error("Never should match nothing")
	}
}

func TestAndEmpty(t *testing.T) {
	p := match.And()
	if !p("anything") {
		t.Error("And with no predicates should match")
	}
}

func TestOrEmpty(t *testing.T) {
	p := match.Or()
	if p("anything") {
		t.Error("Or with no predicates should not match")
	}
}
