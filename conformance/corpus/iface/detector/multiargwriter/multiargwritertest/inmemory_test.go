// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multiargwritertest_test

import (
	"testing"

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
		multiargwritertest.MultiArgWriterModel(),
		multiargwritertest.MultiArgWriterSubject("in-memory", func() multiargwriter.MultiArgWriter {
			return multiargwritertest.NewInMemory()
		}),
	)
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
