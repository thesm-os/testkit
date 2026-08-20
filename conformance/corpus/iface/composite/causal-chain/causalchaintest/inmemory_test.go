// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package causalchaintest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	causalchain "go.thesmos.sh/testkit/conformance/corpus/iface/composite/causal-chain"
	"go.thesmos.sh/testkit/conformance/corpus/iface/composite/causal-chain/causalchaintest"
)

// causal-chain is the model tier's: `AUTO-REPLAY-CAUSAL-ORDERING` walks the
// replay against the dependency graph, whose identifiers and edges are the
// domain's rather than anything derivation can invent.
//
// What the suite tier states is the row below: an effect cannot precede its
// cause. The derived entry leaves the cause list at its zero, which is why a
// log admitting only settled causes accepts it — the dangling entry is the
// row's to build.
func TestLogContract(t *testing.T) {
	t.Parallel()

	causalchaintest.RunLog(t,
		causalchaintest.LogHarness[*causalchaintest.InMemory]{Name: "in-memory", New: causalchaintest.NewInMemory},
		causalchaintest.LogChecks{
			{
				Method: "Append",
				Name:   "refuses-an-unlanded-cause",
				Claim:  "Append refuses an entry whose cause has not landed",
				Run: func(tb testing.TB, s causalchain.Log, fx causalchaintest.LogFixture) {
					tb.Helper()
					dangling := causalchain.Entry{ID: "b6-effect", DependsOn: []string{"b6-cause"}}
					testkit.ErrorIs(tb, s.Append(tb.Context(), dangling),
						causalchaintest.ErrUnmetDependency, "the effect cannot precede its cause")

					testkit.NoError(tb,
						s.Append(tb.Context(), causalchain.Entry{ID: "b6-cause"}),
						"the cause lands first")
					testkit.NoError(tb, s.Append(tb.Context(), dangling),
						"and then the effect is admitted")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestLogContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	causalchaintest.RunLog(t,
		causalchaintest.LogHarness[*causalchaintest.InMemory]{Name: "in-memory", New: causalchaintest.NewInMemory},
		causalchaintest.LogSuite.Without(causalchaintest.LogSuite.Checks.Append.Smoke()),
	)
}
