// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package txtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx/txtest"
)

// seed is the smallest true history this contract has: the suite seeds
// through the classified writers, and both terminal writers refuse a handle
// nothing opened — so the seed opens one and settles it.
func seed(ctx context.Context, subject tx.Contract) error {
	h, err := subject.Begin(ctx)
	if err != nil {
		return err
	}
	return subject.Commit(ctx, h)
}

// tx is the model tier's under ADR-0018: `AUTO-TWO-PHASE-MUTEX` and
// `AUTO-TWO-PHASE-ROLLBACK-AFTER-COMMIT` state it, over the handle Begin
// answers and the terminal pair threads.
//
// What the checks below add is the handle discipline the laws do not draw:
// two open transactions settling independently, and a handle nothing opened
// settling nothing — facts about *which* transaction a terminal operation
// names rather than about the mutex itself.
func TestContractContract(t *testing.T) {
	t.Parallel()

	txtest.AssertContractContract(t,
		txtest.ContractModel(),
		txtest.ContractSubject("in-memory", func() tx.Contract {
			return txtest.NewInMemory()
		}),
		txtest.ContractSeed(seed),
		txtest.ContractOnBegin("settles once and then refuses both terminals", func(
			tb testing.TB, subject tx.Contract,
		) {
			tb.Helper()
			h, err := subject.Begin(tb.Context())
			testkit.NoError(tb, err, "the transaction opens")
			testkit.NoError(tb, subject.Commit(tb.Context(), h), "and commits")

			testkit.ErrorIs(tb, subject.Commit(tb.Context(), h), tx.ErrTxClosed,
				"a second commit has nothing to settle")
			testkit.ErrorIs(tb, subject.Rollback(tb.Context(), h), tx.ErrTxClosed,
				"and a rollback after a commit is the second terminal operation")
		}),
		txtest.ContractOnBegin("rolls back once and then refuses both terminals", func(
			tb testing.TB, subject tx.Contract,
		) {
			tb.Helper()
			// The mirror, because a subject can get one direction right and the
			// other wrong: settling through separate paths gives the rule two
			// places to be written and one place to be forgotten.
			h, err := subject.Begin(tb.Context())
			testkit.NoError(tb, err, "the transaction opens")
			testkit.NoError(tb, subject.Rollback(tb.Context(), h), "and rolls back")

			testkit.ErrorIs(tb, subject.Rollback(tb.Context(), h), tx.ErrTxClosed,
				"a second rollback has nothing to settle")
			testkit.ErrorIs(tb, subject.Commit(tb.Context(), h), tx.ErrTxClosed,
				"and a commit after a rollback is the second terminal operation")
		}),
		txtest.ContractOnBegin("settles two open transactions independently", func(
			tb testing.TB, subject tx.Contract,
		) {
			tb.Helper()
			first, err := subject.Begin(tb.Context())
			testkit.NoError(tb, err, "the first transaction opens")
			second, err := subject.Begin(tb.Context())
			testkit.NoError(tb, err, "and a second opens beside it")

			testkit.NoError(tb, subject.Commit(tb.Context(), first),
				"the first commits")
			testkit.NoError(tb, subject.Rollback(tb.Context(), second),
				"and the second still rolls back — settling one settles one")
		}),
		txtest.ContractOnBegin("refuses a handle that never began", func(
			tb testing.TB, subject tx.Contract,
		) {
			tb.Helper()
			testkit.ErrorIs(tb, subject.Commit(tb.Context(), tx.Tx{ID: 99}), tx.ErrTxClosed,
				"there is nothing to commit under an invented handle")
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
		txtest.ContractSeed(seed),
		txtest.ContractWithout("Begin/smoke"),
		txtest.ContractWithoutDouble(),
	)
}
