// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/spec"
)

type fooPayload struct {
	A int
	B string
}

type barPayload struct {
	X bool
}

func TestAttachment(t *testing.T) {
	t.Parallel()

	t.Run("Set lazily allocates and Get returns the typed value", func(t *testing.T) {
		t.Parallel()
		var m map[string]any
		testkit.False(t, spec.Has(m, "foo"), "Has on nil map")

		spec.Set(&m, "foo", fooPayload{A: 1, B: "x"})
		testkit.True(t, spec.Has(m, "foo"), "Has after Set")

		got, ok := spec.Get[fooPayload](m, "foo")
		testkit.True(t, ok, "Get hits the key")
		testkit.Equal(t, got.A, 1, "field A roundtrip")
		testkit.Equal(t, got.B, "x", "field B roundtrip")
	})

	t.Run("Get on type-mismatched value returns ok=false", func(t *testing.T) {
		t.Parallel()
		var m map[string]any
		spec.Set(&m, "foo", fooPayload{A: 1})
		_, ok := spec.Get[barPayload](m, "foo")
		testkit.False(t, ok, "wrong type returns ok=false")
	})

	t.Run("Get on missing key returns ok=false", func(t *testing.T) {
		t.Parallel()
		m := map[string]any{"foo": fooPayload{}}
		_, ok := spec.Get[fooPayload](m, "bar")
		testkit.False(t, ok, "missing key returns ok=false")
	})

	t.Run("nil-safe operations", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, spec.Has(nil, "any"), "Has(nil)")
		_, ok := spec.Get[fooPayload](nil, "any")
		testkit.False(t, ok, "Get(nil)")
	})
}
