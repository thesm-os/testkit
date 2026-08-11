// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package readyourwritestest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readyourwrites"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readyourwrites/readyourwritestest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	readyourwritestest.AssertMixedContract(t,
		readyourwritestest.MixedSubject("in-memory", func() readyourwrites.Mixed {
			return readyourwritestest.NewInMemory()
		}),
		// The model tier: random sequences against the derived reference,
		// reporting under "model" beside the per-method checks.
		readyourwritestest.MixedModel(),
		readyourwritestest.MixedOnGet("returns what Store wrote", func(
			tb testing.TB, subject readyourwrites.Mixed, key string,
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

	readyourwritestest.AssertMixedContract(t,
		readyourwritestest.MixedSubject("in-memory", func() readyourwrites.Mixed {
			return readyourwritestest.NewInMemory()
		}),
		readyourwritestest.MixedWithoutDouble(),
	)
}
