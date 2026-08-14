// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package monotonicreadstest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonicreads"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonicreads/monotonicreadstest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	monotonicreadstest.AssertMixedContract(t,
		monotonicreadstest.MixedSubject("in-memory", func() monotonicreads.Mixed {
			return monotonicreadstest.NewInMemory()
		}),
		// The model tier: random sequences against the derived reference,
		// reporting under "model/twin" beside the per-method checks — twinned,
		// because no store oracle stamps the version member this subject assigns.
		monotonicreadstest.MixedModel(),
		monotonicreadstest.MixedOnGet("returns what Store wrote", func(
			tb testing.TB, subject monotonicreads.Mixed, key string,
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

	monotonicreadstest.AssertMixedContract(t,
		monotonicreadstest.MixedSubject("in-memory", func() monotonicreads.Mixed {
			return monotonicreadstest.NewInMemory()
		}),
		monotonicreadstest.MixedWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestMixedSaturation(t *testing.T) {
	t.Parallel()
	monotonicreadstest.MixedModelSaturation(t, func() monotonicreads.Mixed {
		return monotonicreadstest.NewInMemory()
	})
}
