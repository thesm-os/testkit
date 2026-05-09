// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package crdttest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/crdt"
	"go.thesmos.sh/testkit/suite"
)

// TestAdditiveCounterContract closes the e2e loop on the
// //testkit:crdt-merge cross-method invariant. The default factory
// returns a fresh additive counter; the suite's AssertCRDTMerge
// subtest constructs two impls, applies (Merge(a), Merge(b)) and
// (Merge(b), Merge(a)) in opposite orders, and asserts state
// equality via the supplied WithStateEqual closure.
//
// reflect.DeepEqual on two *additiveCounter values would reach into
// the embedded sync.Mutex internals; even when both are unlocked
// after the test, the mutex's atomic state words can differ across
// Lock/Unlock cycles. The custom comparator reads the public Value
// instead, which is the only state the CRDT contract cares about.
func TestAdditiveCounterContract(t *testing.T) {
	t.Parallel()
	AssertAdditiveCounterContract(t, func() crdt.AdditiveCounter {
		// Pre-seed sum=42 so Aggregator's "returns expected" baseline
		// (which asserts Value() == 42, the default int sample) is
		// satisfied. The CRDT-merge subtest is order-invariant, so
		// pre-seeded baseline state doesn't bias its convergence
		// check.
		c := crdt.NewAdditiveCounter()
		_ = c.Merge(t.Context(), 42)
		return c
	}, suite.WithStateEqual(func(a, b crdt.AdditiveCounter) bool {
		va, _ := a.Value(t.Context())
		vb, _ := b.Value(t.Context())
		return va == vb
	}))
}
