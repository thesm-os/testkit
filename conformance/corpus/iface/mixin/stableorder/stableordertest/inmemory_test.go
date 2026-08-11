// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stableordertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/stableorder"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/stableorder/stableordertest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	stableordertest.AssertMixedContract(t,
		stableordertest.MixedSubject("in-memory", func() stableorder.Mixed {
			return stableordertest.NewInMemory()
		}),
		stableordertest.MixedOnItems("yields what Add put in, once each", func(
			tb testing.TB, subject stableorder.Mixed,
		) {
			tb.Helper()
			// The suite seeds through Add, so one element is already present.
			// Adding the same key again is what makes the claim testable: a
			// drain that yielded it twice would be reporting its input rather
			// than its contents.
			// A second element, deliberately out of key order: with one
			// element the drain's ordering is unobservable, and a subject
			// that returned map order would pass.
			testkit.NoError(tb, subject.Add(tb.Context(), stableorder.Value{Key: "zz", Body: "last"}),
				"a second element is accepted")
			testkit.NoError(tb, subject.Add(tb.Context(), stableorder.Value{Key: "aa", Body: "first"}),
				"and a third that sorts ahead of it")

			got, err := subject.Items(tb.Context())
			testkit.NoError(tb, err, "the drain succeeds")
			testkit.Equal(tb, len(got), 3, "each append is one element")
			testkit.Equal(tb, got[0].Key, "aa", "and the drain is ordered rather than arbitrary")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	stableordertest.AssertMixedContract(t,
		stableordertest.MixedSubject("in-memory", func() stableorder.Mixed {
			return stableordertest.NewInMemory()
		}),
		stableordertest.MixedWithoutDouble(),
	)
}
