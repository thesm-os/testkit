// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package txtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx/txtest"
)

// tx is owned by no tier under ADR-0018, which the gate reports as a law to
// write rather than a check to invent.
//
// The reason is that the interesting claims need accumulated state — a
// transaction that has already settled — and a generated check receives a fresh
// subject built by the factory. What the extension point can do is accumulate
// that state itself, and everything below does: each check begins from a
// transaction that has not started and drives it wherever it needs to be.
//
// That is not the same as the tier owning the classification. A generated check
// derived from the directive would have to know that `commit=Commit` and
// `rollback=Rollback` are terminal and mutually exclusive, and nothing in the
// contract's declaration says so — the vocabulary names three roles and stops.
func TestContractContract(t *testing.T) {
	t.Parallel()

	txtest.AssertContractContract(t,
		txtest.ContractModel(),
		txtest.ContractSubject("in-memory", func() tx.Contract {
			return txtest.NewInMemory()
		}),
		txtest.ContractOnBegin("settles once and then refuses both terminals", func(
			tb testing.TB, subject tx.Contract,
		) {
			tb.Helper()
			testkit.NoError(tb, subject.Begin(tb.Context()), "the transaction opens")
			testkit.NoError(tb, subject.Commit(tb.Context()), "and commits")

			testkit.ErrorIs(tb, subject.Commit(tb.Context()), txtest.ErrNotOpen,
				"a second commit has nothing to settle")
			testkit.ErrorIs(tb, subject.Rollback(tb.Context()), txtest.ErrNotOpen,
				"and a rollback after a commit is the second terminal operation")
		}),
		txtest.ContractOnBegin("rolls back once and then refuses both terminals", func(
			tb testing.TB, subject tx.Contract,
		) {
			tb.Helper()
			// The mirror, because a subject can get one direction right and the
			// other wrong: settling through separate paths gives the rule two
			// places to be written and one place to be forgotten.
			testkit.NoError(tb, subject.Begin(tb.Context()), "the transaction opens")
			testkit.NoError(tb, subject.Rollback(tb.Context()), "and rolls back")

			testkit.ErrorIs(tb, subject.Rollback(tb.Context()), txtest.ErrNotOpen,
				"a second rollback has nothing to settle")
			testkit.ErrorIs(tb, subject.Commit(tb.Context()), txtest.ErrNotOpen,
				"and a commit after a rollback is the second terminal operation")
		}),
		txtest.ContractOnCommit("refuses a transaction that never began", func(
			tb testing.TB, subject tx.Contract,
		) {
			tb.Helper()
			testkit.ErrorIs(tb, subject.Commit(tb.Context()), txtest.ErrNotOpen,
				"there is nothing to commit")
		}),
		txtest.ContractOnBegin("refuses a second open", func(
			tb testing.TB, subject tx.Contract,
		) {
			tb.Helper()
			testkit.NoError(tb, subject.Begin(tb.Context()), "the transaction opens")
			testkit.ErrorIs(tb, subject.Begin(tb.Context()), txtest.ErrOpen,
				"and a second Begin does not silently replace it")
		}),
		txtest.ContractOnRollback("reopens after settling", func(
			tb testing.TB, subject tx.Contract,
		) {
			tb.Helper()
			// A subject that refused every Begin after the first would pass
			// every check above and be usable exactly once.
			testkit.NoError(tb, subject.Begin(tb.Context()), "the transaction opens")
			testkit.NoError(tb, subject.Rollback(tb.Context()), "and rolls back")
			testkit.NoError(tb, subject.Begin(tb.Context()), "and the handle opens a new one")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	txtest.AssertContractContract(t,
		txtest.ContractSubject("in-memory", func() tx.Contract {
			return txtest.NewInMemory()
		}),
		txtest.ContractWithout("Begin/smoke"),
		txtest.ContractWithoutDouble(),
	)
}
