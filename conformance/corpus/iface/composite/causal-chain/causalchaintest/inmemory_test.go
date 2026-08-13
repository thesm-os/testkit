// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package causalchaintest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	causalchain "go.thesmos.sh/testkit/conformance/corpus/iface/composite/causal-chain"
	"go.thesmos.sh/testkit/conformance/corpus/iface/composite/causal-chain/causalchaintest"
)

// seed is the smallest true history this log has: the suite seeds through
// Append, and a derived entry carries dependencies nothing landed — so the
// seed lands one entry that depends on nothing.
func seed(ctx context.Context, subject causalchain.Log) error {
	return subject.Append(ctx, causalchain.Entry{ID: "seed", Body: "seeded"})
}

// causal-chain is the model tier's: `AUTO-REPLAY-CAUSAL-ORDERING` walks the
// replay against the dependency graph the two doors below spell — the
// identifiers and edges are the domain's, which is why they are supplied
// rather than derived.
func TestLogContract(t *testing.T) {
	t.Parallel()

	causalchaintest.AssertLogContract(t,
		causalchaintest.LogSeed(seed),
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
		causalchaintest.LogSeed(seed),
		causalchaintest.LogSubject("in-memory", func() causalchain.Log {
			return causalchaintest.NewInMemory()
		}),
		causalchaintest.LogWithout("Append/smoke"),
		causalchaintest.LogWithoutDouble(),
	)
}
