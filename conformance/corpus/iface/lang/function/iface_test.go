// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package function_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/function"
)

// These functions exist to be read: the stub generator takes their signatures
// and directives and emits a double, and the generated checks drive the double
// rather than the original. Nothing generated ever calls the real ones.
//
// Their bodies are still real behaviour a reader is invited to believe — the
// fixture says Get reports ErrNotFound for an empty key, and a body that did
// not would make the docblock a lie about the thing being demonstrated. So
// they are checked here, by hand, which is also what a consumer does with the
// functions a double stands in for.
func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("returns the value for a key it was given", func(t *testing.T) {
		t.Parallel()
		got, err := function.Get(t.Context(), "k")
		testkit.NoError(t, err, "a non-empty key is readable")
		testkit.Equal(t, got.Key, "k", "the value carries the key it was asked for")
	})

	t.Run("reports an empty key as not found", func(t *testing.T) {
		t.Parallel()
		// The one branch in the fixture, and the reason it has a sentinel at
		// all — a reader-shaped signature that never fails would not need one.
		_, err := function.Get(t.Context(), "")
		testkit.ErrorIs(t, err, function.ErrNotFound, "an empty key is not found")
	})
}

// TestPut pins the writer half, which carries a mixin rather than a branch.
func TestPut(t *testing.T) {
	t.Parallel()

	t.Run("accepts a value", func(t *testing.T) {
		t.Parallel()
		testkit.NoError(t, function.Put(t.Context(), function.Value{Key: "k", Body: "b"}),
			"a write succeeds")
	})
}

// TestLease pins the contract pair. The two are declared across separate
// functions on purpose, so the partner is resolved by name rather than by
// sharing a receiver.
func TestLease(t *testing.T) {
	t.Parallel()

	t.Run("acquires and releases by name", func(t *testing.T) {
		t.Parallel()
		testkit.NoError(t, function.Acquire(t.Context(), "lease"), "acquiring succeeds")
		testkit.NoError(t, function.Release(t.Context(), "lease"), "releasing succeeds")
	})
}
