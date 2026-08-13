// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stickytest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sticky"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sticky/stickytest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	stickytest.AssertMixedContract(t,
		stickytest.MixedSubject("in-memory", func() sticky.Mixed {
			return stickytest.NewInMemory()
		}),
		// The model tier: random sequences against the derived reference,
		// reporting under "model" beside the per-method checks.
		stickytest.MixedModel(),
		stickytest.MixedOnGet("returns what Store wrote", func(
			tb testing.TB, subject sticky.Mixed, key string,
		) {
			tb.Helper()
			got, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "the seeded key is present")
			testkit.Equal(tb, got.Key, key, "and answers under the key it was stored with")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	stickytest.AssertMixedContract(t,
		stickytest.MixedSubject("in-memory", func() sticky.Mixed {
			return stickytest.NewInMemory()
		}),
		stickytest.MixedWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestMixedSaturation(t *testing.T) {
	t.Parallel()
	stickytest.MixedModelSaturation(t, func() sticky.Mixed {
		return stickytest.NewInMemory()
	})
}
