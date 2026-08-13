// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package associativetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/associative"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/associative/associativetest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	associativetest.AssertMixedContract(t,
		associativetest.MixedModel(),
		associativetest.MixedSubject("in-memory", func() associative.Mixed {
			return associativetest.NewInMemory()
		}),
		associativetest.MixedOnTotal("reports what Apply folded in", func(
			tb testing.TB, subject associative.Mixed,
		) {
			tb.Helper()
			// The suite seeds through Apply, so the fold is non-empty here —
			// a total of zero would be a fold that dropped its input.
			got, err := subject.Total(tb.Context())
			testkit.NoError(tb, err, "the total is readable")
			testkit.NotEqual(tb, got, 0, "and reflects the seeded delta")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	associativetest.AssertMixedContract(t,
		associativetest.MixedSubject("in-memory", func() associative.Mixed {
			return associativetest.NewInMemory()
		}),
		associativetest.MixedWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestMixedSaturation(t *testing.T) {
	t.Parallel()
	associativetest.MixedModelSaturation(t, func() associative.Mixed {
		return associativetest.NewInMemory()
	})
}
