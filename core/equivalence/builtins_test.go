// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package equivalence_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/equivalence"
)

func TestRelationNames(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[entry]()
	cases := []struct {
		got    equivalence.Relation
		prefix string
	}{
		{equivalence.Strict(), "strict"},
		{equivalence.IgnoreFields(typ, "WrittenAt"), "ignore-fields"},
		{equivalence.IgnoreMapKeys(typ, "transient"), "ignore-map-keys"},
		{equivalence.Approximate(typ, "Value", 0.1), "approximate"},
		{equivalence.RegexFields(typ, []string{"ID"}, `^.+$`), "regex-fields"},
		{equivalence.Timestamp(typ, "WrittenAt", time.Second), "timestamp"},
		{equivalence.IDField(typ, "ID"), "id-field"},
		{equivalence.RetryCount(typ, "Attempts"), "retry-count"},
		{equivalence.OrderInvariant(typ, "Items"), "order-invariant"},
		{equivalence.Cardinality(typ, "Items", 0, 5), "cardinality"},
		{equivalence.ErrorClass(reflect.TypeFor[error]()), "error-class"},
		{equivalence.Custom("my-rule", func(_, _ any) bool { return true }), "custom"},
	}

	for _, c := range cases {
		testkit.Assert(t, c.got.Name()).Contains(c.prefix, "relation name")
	}
}

func TestStrict(t *testing.T) {
	t.Parallel()

	r := equivalence.Strict()
	testkit.Equal(t, r.Name(), "strict", "name")
	testkit.Equal(t, len(r.Options()), 0, "no options contributed")

	c := equivalence.NewChain().Add(r)
	testkit.True(t, c.Equal(1, 1), "equal")
	testkit.False(t, c.Equal(1, 2), "not equal")
}

func TestIgnoreFields(t *testing.T) {
	t.Parallel()

	t.Run("drops named fields from comparison", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.IgnoreFields(reflect.TypeFor[entry](), "WrittenAt"))
		a := entry{ID: "k", Value: 1, WrittenAt: time.Unix(1, 0)}
		b := entry{ID: "k", Value: 1, WrittenAt: time.Unix(999, 0)}
		testkit.True(t, c.Equal(a, b), "WrittenAt ignored")
	})

	t.Run("non-ignored fields still compare", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.IgnoreFields(reflect.TypeFor[entry](), "WrittenAt"))
		a := entry{ID: "k", Value: 1}
		b := entry{ID: "k", Value: 2}
		testkit.False(t, c.Equal(a, b), "Value still compared")
	})

	t.Run("name carries type and field list", func(t *testing.T) {
		t.Parallel()
		r := equivalence.IgnoreFields(reflect.TypeFor[entry](), "WrittenAt", "Attempts")
		testkit.Assert(t, r.Name()).
			Contains("ignore-fields", "tag").
			Contains("WrittenAt", "field").
			Contains("Attempts", "second field")
	})
}

func TestIgnoreMapKeys(t *testing.T) {
	t.Parallel()

	type mapHolder struct {
		Items map[string]any
	}

	t.Run("drops named keys from map comparison", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.IgnoreMapKeys(reflect.TypeFor[mapHolder](), "transient"))
		a := mapHolder{Items: map[string]any{"k": 1, "transient": "a"}}
		b := mapHolder{Items: map[string]any{"k": 1, "transient": "b"}}
		testkit.True(t, c.Equal(a, b), "transient key dropped")
	})
}

func TestApproximate(t *testing.T) {
	t.Parallel()

	type measure struct {
		Reading float64
		Tag     string
	}

	t.Run("close values within tolerance are equal", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.Approximate(reflect.TypeFor[measure](), "Reading", 0.001))
		a := measure{Reading: 1.0001, Tag: "x"}
		b := measure{Reading: 1.0002, Tag: "x"}
		testkit.True(t, c.Equal(a, b), "within tolerance")
	})

	t.Run("far values diverge", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.Approximate(reflect.TypeFor[measure](), "Reading", 0.001))
		a := measure{Reading: 1.0}
		b := measure{Reading: 2.0}
		testkit.False(t, c.Equal(a, b), "outside tolerance")
	})

	t.Run("non-target field still compared strictly", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.Approximate(reflect.TypeFor[measure](), "Reading", 0.001))
		a := measure{Reading: 1.0, Tag: "x"}
		b := measure{Reading: 1.0, Tag: "y"}
		testkit.False(t, c.Equal(a, b), "Tag mismatch caught")
	})
}

func TestRegexFields(t *testing.T) {
	t.Parallel()

	type token struct {
		Value string
	}

	t.Run("both values matching the pattern are equal", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().Add(equivalence.RegexFields(
			reflect.TypeFor[token](), []string{"Value"}, `^[a-f0-9]{16}$`,
		))
		a := token{Value: "deadbeefcafebabe"}
		b := token{Value: "0123456789abcdef"}
		testkit.True(t, c.Equal(a, b), "both match pattern")
	})

	t.Run("non-matching values diverge", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().Add(equivalence.RegexFields(
			reflect.TypeFor[token](), []string{"Value"}, `^[a-f0-9]{16}$`,
		))
		a := token{Value: "deadbeefcafebabe"}
		b := token{Value: "not-hex"}
		testkit.False(t, c.Equal(a, b), "second doesn't match")
	})
}

func TestTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("times within tolerance are equal", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.Timestamp(reflect.TypeFor[entry](), "WrittenAt", time.Second))
		a := entry{ID: "k", WrittenAt: time.Unix(1000, 0)}
		b := entry{ID: "k", WrittenAt: time.Unix(1000, 500_000_000)}
		testkit.True(t, c.Equal(a, b), "0.5s within 1s tolerance")
	})

	t.Run("times outside tolerance diverge", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.Timestamp(reflect.TypeFor[entry](), "WrittenAt", time.Second))
		a := entry{ID: "k", WrittenAt: time.Unix(1000, 0)}
		b := entry{ID: "k", WrittenAt: time.Unix(1010, 0)}
		testkit.False(t, c.Equal(a, b), "10s outside 1s tolerance")
	})
}

func TestIDField(t *testing.T) {
	t.Parallel()

	t.Run("any non-empty ID equals any other", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.IDField(reflect.TypeFor[entry](), "ID"))
		a := entry{ID: "abc", Value: 1}
		b := entry{ID: "xyz", Value: 1}
		testkit.True(t, c.Equal(a, b), "different IDs equal when both non-empty")
	})

	t.Run("empty vs non-empty diverge", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.IDField(reflect.TypeFor[entry](), "ID"))
		a := entry{ID: "abc"}
		b := entry{ID: ""}
		testkit.False(t, c.Equal(a, b), "empty must not equal non-empty")
	})
}

func TestRetryCount(t *testing.T) {
	t.Parallel()

	t.Run("zero equals zero", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.RetryCount(reflect.TypeFor[entry](), "Attempts"))
		testkit.True(t, c.Equal(entry{Attempts: 0}, entry{Attempts: 0}), "zero=zero")
	})

	t.Run("positive equals positive", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.RetryCount(reflect.TypeFor[entry](), "Attempts"))
		testkit.True(t, c.Equal(entry{Attempts: 1}, entry{Attempts: 5}), "any-positive=any-positive")
	})

	t.Run("zero vs positive diverge", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.RetryCount(reflect.TypeFor[entry](), "Attempts"))
		testkit.False(t, c.Equal(entry{Attempts: 0}, entry{Attempts: 1}), "zero != positive")
	})
}

func TestOrderInvariant(t *testing.T) {
	t.Parallel()

	type bag struct {
		Items []string
	}

	t.Run("same elements in different order are equal", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.OrderInvariant(reflect.TypeFor[bag](), "Items"))
		a := bag{Items: []string{"x", "y", "z"}}
		b := bag{Items: []string{"z", "x", "y"}}
		testkit.True(t, c.Equal(a, b), "same multiset")
	})

	t.Run("different elements diverge", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.OrderInvariant(reflect.TypeFor[bag](), "Items"))
		a := bag{Items: []string{"x", "y"}}
		b := bag{Items: []string{"x", "z"}}
		testkit.False(t, c.Equal(a, b), "different elements")
	})
}

func TestCardinality(t *testing.T) {
	t.Parallel()

	type bag struct {
		Items []int
	}

	t.Run("both within range are equal regardless of contents", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.Cardinality(reflect.TypeFor[bag](), "Items", 1, 5))
		a := bag{Items: []int{1, 2}}
		b := bag{Items: []int{99, 100, 101}}
		testkit.True(t, c.Equal(a, b), "both in [1,5]")
	})

	t.Run("one outside range diverges", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.Cardinality(reflect.TypeFor[bag](), "Items", 1, 5))
		a := bag{Items: []int{1, 2}}
		b := bag{Items: []int{1, 2, 3, 4, 5, 6}}
		testkit.False(t, c.Equal(a, b), "second too large")
	})
}

type myError struct{ msg string }

func (e *myError) Error() string { return e.msg }

type otherError struct{ msg string }

func (e *otherError) Error() string { return e.msg }

func TestErrorClass(t *testing.T) {
	t.Parallel()

	t.Run("same class equates regardless of message", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.ErrorClass(reflect.TypeFor[*myError]()))
		a := error(&myError{msg: "boom"})
		b := error(&myError{msg: "kaboom"})
		testkit.True(t, c.Equal(a, b), "same class equates")
	})

	t.Run("different classes diverge", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.ErrorClass(reflect.TypeFor[*myError]()))
		a := error(&myError{msg: "boom"})
		b := error(&otherError{msg: "boom"})
		testkit.False(t, c.Equal(a, b), "different classes diverge")
	})

	t.Run("nil and nil equate", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.ErrorClass(reflect.TypeFor[*myError]()))
		var a, b error
		testkit.True(t, c.Equal(a, b), "nil=nil")
	})

	t.Run("nil vs non-nil diverges", func(t *testing.T) {
		t.Parallel()
		c := equivalence.NewChain().
			Add(equivalence.ErrorClass(reflect.TypeFor[*myError]()))
		var a error
		b := error(&myError{msg: "boom"})
		testkit.False(t, c.Equal(a, b), "nil!=non-nil")
	})

	t.Run("interface error type matches via Implements", func(t *testing.T) {
		t.Parallel()
		errIface := reflect.TypeFor[error]()
		c := equivalence.NewChain().Add(equivalence.ErrorClass(errIface))
		a := error(errors.New("a"))
		b := error(&myError{msg: "b"})
		testkit.True(t, c.Equal(a, b), "both implement error")
	})

	t.Run("non-error pairs fall through to default equality", func(t *testing.T) {
		t.Parallel()
		// ErrorClass's FilterValues skips pairs that aren't errors,
		// letting the chain fall through to deep equality. Verifies
		// the relation doesn't incorrectly equate non-errors.
		c := equivalence.NewChain().
			Add(equivalence.ErrorClass(reflect.TypeFor[error]()))
		testkit.True(t, c.Equal(42, 42), "non-errors equal under deep equality")
		testkit.False(t, c.Equal(42, 43), "non-errors diverge under deep equality")
	})
}

func TestCustom(t *testing.T) {
	t.Parallel()

	t.Run("uses consumer-supplied function", func(t *testing.T) {
		t.Parallel()
		// Always-true comparator — every value pair becomes equal.
		c := equivalence.NewChain().
			Add(equivalence.Custom("always", func(_, _ any) bool { return true }))
		testkit.True(t, c.Equal(1, 2), "always-true")
	})

	t.Run("name carries the consumer tag", func(t *testing.T) {
		t.Parallel()
		r := equivalence.Custom("my-rule", func(_, _ any) bool { return true })
		testkit.Equal(t, r.Name(), "custom:my-rule", "name")
	})
}

func TestChainComposition(t *testing.T) {
	t.Parallel()

	// Realistic migration-grade chain: drop WrittenAt, treat IDs as
	// opaque, treat Attempts as zero-or-positive, strict elsewhere.
	c := equivalence.NewChain().
		Add(equivalence.IgnoreFields(reflect.TypeFor[entry](), "WrittenAt")).
		Add(equivalence.IDField(reflect.TypeFor[entry](), "ID")).
		Add(equivalence.RetryCount(reflect.TypeFor[entry](), "Attempts"))

	t.Run("real-world entries with legitimate variation are equal", func(t *testing.T) {
		t.Parallel()
		a := entry{ID: "abc", Value: 42, WrittenAt: time.Unix(1, 0), Attempts: 1}
		b := entry{ID: "xyz", Value: 42, WrittenAt: time.Unix(999, 0), Attempts: 5}
		testkit.True(t, c.Equal(a, b), "all variation is configured-equivalent")
	})

	t.Run("Value mismatch still caught", func(t *testing.T) {
		t.Parallel()
		a := entry{ID: "abc", Value: 42}
		b := entry{ID: "xyz", Value: 99}
		testkit.False(t, c.Equal(a, b), "Value diverges")
	})
}
