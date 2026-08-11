// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package snapshotisolationtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/snapshotisolation"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/snapshotisolation/snapshotisolationtest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	snapshotisolationtest.AssertMixedContract(t,
		snapshotisolationtest.MixedSubject("in-memory", func() snapshotisolation.Mixed {
			return snapshotisolationtest.NewInMemory()
		}),
		snapshotisolationtest.MixedOnHistory("hands back a copy, not the backing array", func(
			tb testing.TB, subject snapshotisolation.Mixed,
		) {
			tb.Helper()
			// The suite seeds through Record, so the history is non-empty.
			// Mutating what History returned must not reach the subject: an
			// anomaly check walks this while the subject may still record.
			got, err := subject.History(tb.Context())
			testkit.NoError(tb, err, "the history is readable")
			testkit.Equal(tb, len(got), 1, "the seeded entry is there")

			got[0].Txn = 99
			again, err := subject.History(tb.Context())
			testkit.NoError(tb, err, "the history is still readable")
			testkit.NotEqual(tb, again[0].Txn, 99, "the caller's edit did not reach the subject")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	snapshotisolationtest.AssertMixedContract(t,
		snapshotisolationtest.MixedSubject("in-memory", func() snapshotisolation.Mixed {
			return snapshotisolationtest.NewInMemory()
		}),
		snapshotisolationtest.MixedWithoutDouble(),
	)
}
