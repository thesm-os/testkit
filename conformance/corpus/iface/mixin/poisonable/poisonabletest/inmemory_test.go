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

	poisonabletest.RunMixed(t,
		poisonabletest.MixedHarness[*poisonabletest.InMemory]{Name: "in-memory", New: poisonabletest.NewInMemory},
		poisonabletest.MixedChecks{
			{
				Method: "Probe",
				Name:   "latches-the-state-it-was-driven-into",
				Claim:  "Probe keeps reporting the state it was driven into",
				Run: func(tb testing.TB, s poisonable.Mixed, fx poisonabletest.MixedFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Probe(), "a fresh subject is healthy")
					testkit.NoError(tb, s.Fail(tb.Context()), "and can be driven to fail")

					testkit.Error(tb, s.Probe(), "the failure is reported")
					testkit.Error(tb, s.Probe(), "and reading it does not clear it")
				},
			},
		},
	)
}
