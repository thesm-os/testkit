// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package domhint_test

import (
	"reflect"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/domhint"
)

// runID is a fixture opaque-domain type.
type runID string

// fenceToken is a second fixture so collision and multi-type tests
// have a distinct second registration.
type fenceToken int64

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("stores generator under T's reflect.Type", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		gen := rapid.Just(runID("seed"))
		domhint.Register(r, "ids.RunID", gen)
		got, ok := domhint.Lookup[runID](r)
		testkit.True(t, ok, "registered hint resolves")
		testkit.True(t, got == gen, "same generator pointer returned")
	})

	t.Run("empty name defaults to type qualified string", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		domhint.Register(r, "", rapid.Just(runID("x")))
		_, name, ok := domhint.LookupByType(r, reflect.TypeFor[runID]())
		testkit.True(t, ok, "registered")
		testkit.True(t, name != "", "non-empty default name")
	})

	t.Run("collision panics with diagnostic", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		domhint.Register(r, "ids.RunID", rapid.Just(runID("a")))
		recovered := testkit.Panics(t, func() {
			domhint.Register(r, "ids.RunID", rapid.Just(runID("b")))
		}, "collision must panic")
		testkit.Assert(t, asString(recovered)).
			Contains("already registered", "diagnostic mentions duplicate").
			Contains("ids.RunID", "diagnostic names the type")
	})
}

func TestLookup(t *testing.T) {
	t.Parallel()

	t.Run("unregistered type yields false", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		_, ok := domhint.Lookup[runID](r)
		testkit.False(t, ok, "absent")
	})

	t.Run("nil registry yields false", func(t *testing.T) {
		t.Parallel()
		_, ok := domhint.Lookup[runID](nil)
		testkit.False(t, ok, "nil")
	})

	t.Run("multiple types register independently", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		domhint.Register(r, "ids.RunID", rapid.Just(runID("r")))
		domhint.Register(r, "ids.FenceToken", rapid.Just(fenceToken(42)))
		_, okA := domhint.Lookup[runID](r)
		_, okB := domhint.Lookup[fenceToken](r)
		testkit.True(t, okA, "RunID present")
		testkit.True(t, okB, "FenceToken present")
	})
}

func TestLookupByType(t *testing.T) {
	t.Parallel()

	t.Run("returns typed hint plus display name", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		domhint.Register(r, "ids.RunID", rapid.Just(runID("x")))
		got, name, ok := domhint.LookupByType(r, reflect.TypeFor[runID]())
		testkit.True(t, ok, "found")
		testkit.Equal(t, name, "ids.RunID", "name preserved")
		_, isHint := got.(domhint.Hint[runID])
		testkit.True(t, isHint, "stored as Hint[T]")
	})

	t.Run("absent type yields false", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		_, _, ok := domhint.LookupByType(r, reflect.TypeFor[runID]())
		testkit.False(t, ok, "absent")
	})

	t.Run("nil registry yields false", func(t *testing.T) {
		t.Parallel()
		_, _, ok := domhint.LookupByType(nil, reflect.TypeFor[runID]())
		testkit.False(t, ok, "nil")
	})
}

func TestTypeNamesAndLen(t *testing.T) {
	t.Parallel()

	t.Run("TypeNames returns sorted display names", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		domhint.Register(r, "zebra", rapid.Just(runID("z")))
		domhint.Register(r, "alpha", rapid.Just(fenceToken(0)))
		got := domhint.TypeNames(r)
		testkit.Equal(t, got, []string{"alpha", "zebra"}, "sorted")
	})

	t.Run("Len counts unique types", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		testkit.Equal(t, domhint.Len(r), 0, "empty")
		domhint.Register(r, "", rapid.Just(runID("x")))
		testkit.Equal(t, domhint.Len(r), 1, "one")
	})

	t.Run("nil registry yields empty slice and zero", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, domhint.TypeNames(nil) == nil, "nil names")
		testkit.Equal(t, domhint.Len(nil), 0, "nil len")
	})
}

func TestDefaultRegistry(t *testing.T) {
	t.Parallel()

	t.Run("returns the same singleton across calls", func(t *testing.T) {
		t.Parallel()
		a := domhint.DefaultRegistry()
		b := domhint.DefaultRegistry()
		testkit.True(t, a == b, "singleton")
	})
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case error:
		return x.Error()
	default:
		return ""
	}
}
