// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package atomictest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/atomic"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/atomic/atomictest"
)

// atomic is the model tier's under ADR-0018 — AUTO-ATOMIC-WRITE states it,
// comparing observable state around the write the subject refuses.
//
// The rows below are the deterministic complement: one accepted entry read
// back whole, and one refused entry that landed nowhere.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	atomictest.RunMixed(t,
		atomictest.MixedHarness[*atomictest.InMemory]{Name: "in-memory", New: atomictest.NewInMemory},
		atomictest.MixedChecks{
			{
				Method: "Read",
				Name:   "returns-the-whole-entry",
				Claim:  "Read returns the whole entry as it was written",
				Run: func(tb testing.TB, s atomic.Mixed, fx atomictest.MixedFixture) {
					tb.Helper()
					e := atomic.Entry{Key: fx.Key(), Left: "left", Right: "right"}
					testkit.NoError(tb, s.Write(tb.Context(), e), "a whole entry lands")

					got, err := s.Read(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a written key is found")
					testkit.Equal(tb, got, e, "and carries both halves")
				},
			},
			{
				Method: "Read",
				Name:   "half-an-entry-lands-nowhere",
				Claim:  "Read refuses half an entry whole",
				Run: func(tb testing.TB, s atomic.Mixed, fx atomictest.MixedFixture) {
					tb.Helper()
					half := atomic.Entry{Key: fx.KeyOther(), Left: "only"}
					testkit.ErrorIs(tb, s.Write(tb.Context(), half), atomictest.ErrHalfEntry,
						"an entry missing one half is refused")

					_, err := s.Read(tb.Context(), fx.KeyOther())
					testkit.ErrorIs(tb, err, atomictest.ErrNotFound, "and nothing landed")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	atomictest.RunMixed(t,
		atomictest.MixedHarness[*atomictest.InMemory]{Name: "in-memory", New: atomictest.NewInMemory},
		atomictest.MixedSuite.Without(atomictest.MixedSuite.Checks.Write.Smoke()),
	)
}
