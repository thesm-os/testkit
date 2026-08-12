// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package poisonabletest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/poisonable"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/poisonable/poisonabletest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	poisonabletest.AssertMixedContract(t,
		poisonabletest.MixedModel(),
		poisonabletest.MixedSubject("in-memory", func() poisonable.Mixed {
			return poisonabletest.NewInMemory()
		}),
		poisonabletest.MixedOnProbe("keeps reporting the state it was driven into", func(
			tb testing.TB, subject poisonable.Mixed,
		) {
			tb.Helper()
			testkit.NoError(tb, subject.Probe(), "a fresh subject is healthy")
			testkit.NoError(tb, subject.Fail(tb.Context()), "and can be driven to fail")

			testkit.Error(tb, subject.Probe(), "the failure is reported")
			testkit.Error(tb, subject.Probe(), "and reading it does not clear it")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	poisonabletest.AssertMixedContract(t,
		poisonabletest.MixedSubject("in-memory", func() poisonable.Mixed {
			return poisonabletest.NewInMemory()
		}),
		poisonabletest.MixedWithoutDouble(),
	)
}
