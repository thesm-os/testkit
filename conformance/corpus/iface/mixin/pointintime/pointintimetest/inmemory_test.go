// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package pointintimetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pointintime"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pointintime/pointintimetest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	pointintimetest.AssertMixedContract(t,
		pointintimetest.MixedSubject("in-memory", func() pointintime.Mixed {
			return pointintimetest.NewInMemory()
		}),
		// The model tier: random sequences against the derived reference,
		// reporting under "model" beside the per-method checks.
		pointintimetest.MixedModel(),
		pointintimetest.MixedOnGet("returns what Store wrote", func(
			tb testing.TB, subject pointintime.Mixed, key string,
		) {
			tb.Helper()
			// The suite seeds through Store, so the key is already present —
			// which is what makes this a statement about the pair rather than
			// about Get alone.
			got, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "the seeded key is present")
			testkit.Equal(tb, got.Key, key, "and Get answers under the key it was stored with")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	pointintimetest.AssertMixedContract(t,
		pointintimetest.MixedSubject("in-memory", func() pointintime.Mixed {
			return pointintimetest.NewInMemory()
		}),
		pointintimetest.MixedWithoutDouble(),
	)
}
