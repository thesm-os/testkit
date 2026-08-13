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

	chaintest.AssertContractContract(t,
		chaintest.ContractModel(),
		chaintest.ContractSubject("in-memory", func() chain.Contract {
			return chaintest.NewInMemory()
		}),
		chaintest.ContractOnReplay("yields what Append put in, in order", func(
			tb testing.TB, subject chain.Contract,
		) {
			tb.Helper()
			// The suite seeds through Append, so the log is non-empty. The
			// pairing is the contract: an append with no replay states
			// nothing, which is why eidos requires the partner.
			got, err := subject.Replay(tb.Context())
			testkit.NoError(tb, err, "the log replays")
			testkit.Equal(tb, len(got), 1, "the seeded entry is there")
			testkit.NoError(tb, subject.Verify(tb.Context()), "and the chain verifies")

			// The lever: an entry recorded without extending the digest is
			// the divergence a tampered log has, and it is the only way to
			// reach the verify role's failure arm through the interface.
			testkit.NoError(tb, subject.Append(tb.Context(),
				chain.Entry{Key: chaintest.BreakKey, Body: "unlinked"}),
				"the entry is recorded")
			testkit.ErrorIs(tb, subject.Verify(tb.Context()), chaintest.ErrBroken,
				"and the break is detected rather than served")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	chaintest.AssertContractContract(t,
		chaintest.ContractSubject("in-memory", func() chain.Contract {
			return chaintest.NewInMemory()
		}),
		chaintest.ContractWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	chaintest.ContractModelSaturation(t, func() chain.Contract {
		return chaintest.NewInMemory()
	})
}
