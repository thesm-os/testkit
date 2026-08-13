// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package causaltest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/causal"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/causal/causaltest"
	"go.thesmos.sh/testkit/engine/model/law"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	causaltest.AssertMixedContract(t,
		causaltest.MixedSubject("in-memory", func() causal.Mixed {
			return causaltest.NewInMemory()
		}),
		// The model tier: random sequences against the twin reference,
		// reporting under "model/twin" beside the per-method checks.
		causaltest.MixedModel(
			// The happens-before door: this store's causal order is the
			// revision order within one key — an earlier same-key write is
			// the cause every later observation of that key must respect.
			causaltest.MixedModelHappensBefore(func(a, b law.ClientOp[string]) bool {
				return a.Write && a.Key == b.Key && a.Version < b.Version
			}),
		),
		causaltest.MixedOnGet("returns what Store wrote", func(
			tb testing.TB, subject causal.Mixed, key string,
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

	causaltest.AssertMixedContract(t,
		causaltest.MixedSubject("in-memory", func() causal.Mixed {
			return causaltest.NewInMemory()
		}),
		causaltest.MixedWithoutDouble(),
	)
}
