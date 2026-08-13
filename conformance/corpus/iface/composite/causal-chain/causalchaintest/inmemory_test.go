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
// replay against the dependency graph the two doors below spell — the
// identifiers and edges are the domain's, which is why they are supplied
// rather than derived.
//
// The seed is not supplied, and that is the assertion. The derived one lands
// the fixture's own entry through Append, which a log admitting only settled
// causes accepts precisely because the derivation leaves the cause list at its
// zero. Both call sites here once passed a hand-written seed to work around a
// derived entry that named a cause nothing had landed.
func TestLogContract(t *testing.T) {
	t.Parallel()

	causalchaintest.AssertLogContract(t,
		causalchaintest.LogModel(
			causalchaintest.LogModelEntryID(func(e causalchain.Entry) string { return e.ID }),
			causalchaintest.LogModelDependsOn(func(e causalchain.Entry) []string { return e.DependsOn }),
		),
		causalchaintest.LogSubject("in-memory", func() causalchain.Log {
			return causalchaintest.NewInMemory()
		}),
		causalchaintest.LogOnAppend("refuses an entry whose cause has not landed", func(
			tb testing.TB, subject causalchain.Log, _ causalchain.Entry,
		) {
			tb.Helper()
			dangling := causalchain.Entry{ID: "b6-effect", DependsOn: []string{"b6-cause"}}
			testkit.ErrorIs(tb, subject.Append(tb.Context(), dangling),
				causalchaintest.ErrUnmetDependency, "the effect cannot precede its cause")

			testkit.NoError(tb,
				subject.Append(tb.Context(), causalchain.Entry{ID: "b6-cause"}),
				"the cause lands first")
			testkit.NoError(tb, subject.Append(tb.Context(), dangling),
				"and then the effect is admitted")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestLogContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	causalchaintest.AssertLogContract(t,
		causalchaintest.LogSubject("in-memory", func() causalchain.Log {
			return causalchaintest.NewInMemory()
		}),
		causalchaintest.LogWithout("Append/smoke"),
		causalchaintest.LogWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// with the same doors armed that arm the tier.
func TestLogSaturation(t *testing.T) {
	t.Parallel()
	causalchaintest.LogModelSaturation(t, func() causalchain.Log {
		return causalchaintest.NewInMemory()
	},
		causalchaintest.LogModelEntryID(func(e causalchain.Entry) string { return e.ID }),
		causalchaintest.LogModelDependsOn(func(e causalchain.Entry) []string { return e.DependsOn }),
	)
}
