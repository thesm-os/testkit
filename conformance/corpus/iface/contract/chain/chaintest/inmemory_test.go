// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package chaintest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/chain"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/chain/chaintest"
)

// The generated contract, run against the in-memory subject.
func TestContractContract(t *testing.T) {
	t.Parallel()

	chaintest.RunContract(t,
		chaintest.ContractHarness[*chaintest.InMemory]{Name: "in-memory", New: chaintest.NewInMemory},
		chaintest.ContractChecks{
			{
				Method: "Replay",
				Name:   "yields-appends-in-order",
				Claim:  "Replay yields what Append put in, in order",
				Run: func(tb testing.TB, s chain.Contract, fx chaintest.ContractFixture) {
					tb.Helper()
					// The pairing is the contract: an append with no replay
					// states nothing, which is why eidos requires the partner.
					testkit.NoError(tb, s.Append(tb.Context(), fx.Entry()), "the entry is recorded")

					got, err := s.Replay(tb.Context())
					testkit.NoError(tb, err, "the log replays")
					testkit.Equal(tb, len(got), 1, "the appended entry is there")
					testkit.NoError(tb, s.Verify(tb.Context()), "and the chain verifies")

					// The lever: an entry recorded without extending the digest
					// is the divergence a tampered log has, and it is the only
					// way to reach the verify role's failure arm through the
					// interface.
					testkit.NoError(tb, s.Append(tb.Context(),
						chain.Entry{Key: chaintest.BreakKey, Body: "unlinked"}),
						"the unlinked entry is recorded")
					testkit.ErrorIs(tb, s.Verify(tb.Context()), chaintest.ErrBroken,
						"and the break is detected rather than served")
				},
			},
		},
	)
}
