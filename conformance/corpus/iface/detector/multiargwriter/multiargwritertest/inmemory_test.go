// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multiargwritertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multiargwriter"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multiargwriter/multiargwritertest"
)

// A three-argument write seeds itself, which it did not always.
//
// The seed derivation matched the single-argument `writer` alone, so the
// ordinary keyed store — the commonest write there is — could not populate its
// own subject and every consumer wrote the hook by hand. The three writer
// detectors differ in arity and nothing else, and the seed passes whatever the
// method declares, so arity was never something it had to know.
func TestMultiArgWriterContract(t *testing.T) {
	t.Parallel()

	multiargwritertest.AssertMultiArgWriterContract(t,
		multiargwritertest.MultiArgWriterSubject("in-memory", func() multiargwriter.MultiArgWriter {
			return multiargwritertest.NewInMemory()
		}),
	)
}

// That the write lands under the key it was given, which the signature cannot
// say: Set reports whether it failed, never what it stored.
func TestSetStoresUnderItsKey(t *testing.T) {
	t.Parallel()

	s := multiargwritertest.NewInMemory()
	testkit.NoError(t, s.Set(t.Context(), "k", "b"), "storing succeeds")

	got, ok := s.Stored("k")
	testkit.True(t, ok, "the value is found under the key it was given")
	testkit.Equal(t, got, "b", "and carries what was written")
}

// Declining the double is separate from dropping a check.
func TestMultiArgWriterContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	multiargwritertest.AssertMultiArgWriterContract(t,
		multiargwritertest.MultiArgWriterSubject("in-memory", func() multiargwriter.MultiArgWriter {
			return multiargwritertest.NewInMemory()
		}),
		multiargwritertest.MultiArgWriterWithout("Set/smoke"),
		multiargwritertest.MultiArgWriterWithoutDouble(),
	)
}
