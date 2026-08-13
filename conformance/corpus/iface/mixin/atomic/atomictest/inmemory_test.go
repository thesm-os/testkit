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
// The checks below are the deterministic complement: one accepted entry read
// back whole, and one refused entry that landed nowhere.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	atomictest.AssertMixedContract(t,
		atomictest.MixedModel(),
		atomictest.MixedSubject("in-memory", func() atomic.Mixed {
			return atomictest.NewInMemory()
		}),
		atomictest.MixedOnRead("returns the whole entry as it was written", func(
			tb testing.TB, subject atomic.Mixed, key string,
		) {
			tb.Helper()
			e := atomic.Entry{Key: key, Left: "left", Right: "right"}
			testkit.NoError(tb, subject.Write(tb.Context(), e), "a whole entry lands")

			got, err := subject.Read(tb.Context(), key)
			testkit.NoError(tb, err, "a written key is found")
			testkit.Equal(tb, got, e, "and carries both halves")
		}),
		atomictest.MixedOnRead("refuses half an entry whole", func(
			tb testing.TB, subject atomic.Mixed, _ string,
		) {
			tb.Helper()
			half := atomic.Entry{Key: "b6-half", Left: "only"}
			testkit.ErrorIs(tb, subject.Write(tb.Context(), half), atomictest.ErrHalfEntry,
				"an entry missing one half is refused")

			_, err := subject.Read(tb.Context(), "b6-half")
			testkit.ErrorIs(tb, err, atomictest.ErrNotFound, "and nothing landed")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	atomictest.AssertMixedContract(t,
		atomictest.MixedSubject("in-memory", func() atomic.Mixed {
			return atomictest.NewInMemory()
		}),
		atomictest.MixedWithout("Write/smoke"),
		atomictest.MixedWithoutDouble(),
	)
}
