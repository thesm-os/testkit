// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package transactiontest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/transaction"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/transaction/transactiontest"
)

// transaction is the model tier's under ADR-0018: `AUTO-TRANSACTION-ROLLBACK`
// and `AUTO-TRANSACTION-NO-MID-TX-VISIBILITY` state it.
//
// Both need a call that fails on demand, which is exactly what a fixed sequence
// against one subject cannot arrange. The suite tier gets the family that needs
// no failure — that a unit of work survives a derived key, and that a cancelled
// caller's work is refused rather than half-applied.
func TestContractContract(t *testing.T) {
	t.Parallel()

	transactiontest.AssertContractContract(t,
		transactiontest.ContractModel(),
		transactiontest.ContractSubject("in-memory", func() transaction.Contract {
			return transactiontest.NewInMemory()
		}),
		transactiontest.ContractOnRun("reports a unit of work that fails partway", func(
			tb testing.TB, subject transaction.Contract, key string,
		) {
			tb.Helper()
			// The lever is the key, so the failure is reachable through the
			// interface. What is *not* reachable is the claim the law makes —
			// that the staged half left nothing behind — because the contract
			// declares no way to observe the state. That is why the
			// classification is the model tier's rather than this one's.
			testkit.ErrorIs(tb, subject.Run(tb.Context(), transactiontest.DoomedKey),
				transactiontest.ErrDoomed, "a doomed unit of work reports its failure")
			testkit.NoError(tb, subject.Run(tb.Context(), key),
				"and leaves the store usable for the next one")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	transactiontest.AssertContractContract(t,
		transactiontest.ContractSubject("in-memory", func() transaction.Contract {
			return transactiontest.NewInMemory()
		}),
		transactiontest.ContractWithout("Run/smoke"),
		transactiontest.ContractWithoutDouble(),
	)
}
