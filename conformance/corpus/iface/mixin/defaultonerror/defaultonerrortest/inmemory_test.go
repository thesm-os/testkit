// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaultonerrortest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/defaultonerror"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/defaultonerror/defaultonerrortest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	defaultonerrortest.AssertMixedContract(t,
		defaultonerrortest.MixedSubject("in-memory", func() defaultonerror.Mixed {
			return defaultonerrortest.NewInMemory()
		}),
		defaultonerrortest.MixedOnGet("returns what Store wrote", func(
			tb testing.TB, subject defaultonerror.Mixed, key string,
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

	defaultonerrortest.AssertMixedContract(t,
		defaultonerrortest.MixedSubject("in-memory", func() defaultonerror.Mixed {
			return defaultonerrortest.NewInMemory()
		}),
		defaultonerrortest.MixedWithoutDouble(),
	)
}
