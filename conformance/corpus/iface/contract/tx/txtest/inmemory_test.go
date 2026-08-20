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

// seed is the smallest true history this contract has: both terminal writers
// refuse a handle nothing opened, so the seed opens one and settles it.
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

	txtest.RunContract(t,
		txtest.ContractHarness[*txtest.InMemory]{
			Name: "in-memory",
			// The smallest true history this contract has: both terminal
			// writers refuse a handle nothing opened, so the constructor opens
			// one and settles it.
			New: func() *txtest.InMemory {
				s := txtest.NewInMemory()
				if err := seed(t.Context(), s); err != nil {
					panic("txtest_test: seeding: " + err.Error())
				}
				return s
			},
		},
		txtest.ContractChecks{
			{
				Method: "Begin",
				Name:   "settles-once-after-a-commit",
				Claim:  "Begin settles once and then refuses both terminals",
				Run: func(tb testing.TB, s tx.Contract, fx txtest.ContractFixture) {
					tb.Helper()
					h, err := s.Begin(tb.Context())
					testkit.NoError(tb, err, "the transaction opens")
					testkit.NoError(tb, s.Commit(tb.Context(), h), "and commits")

					testkit.ErrorIs(tb, s.Commit(tb.Context(), h), tx.ErrTxClosed,
						"a second commit has nothing to settle")
					testkit.ErrorIs(tb, s.Rollback(tb.Context(), h), tx.ErrTxClosed,
						"and a rollback after a commit is the second terminal operation")
				},
			},
			{
				Method: "Begin",
				Name:   "settles-once-after-a-rollback",
				Claim:  "Begin rolls back once and then refuses both terminals",
				Run: func(tb testing.TB, s tx.Contract, fx txtest.ContractFixture) {
					tb.Helper()
					// The mirror, because a subject can get one direction right
					// and the other wrong: settling through separate paths gives
					// the rule two places to be written and one to be forgotten.
					h, err := s.Begin(tb.Context())
					testkit.NoError(tb, err, "the transaction opens")
					testkit.NoError(tb, s.Rollback(tb.Context(), h), "and rolls back")

					testkit.ErrorIs(tb, s.Rollback(tb.Context(), h), tx.ErrTxClosed,
						"a second rollback has nothing to settle")
					testkit.ErrorIs(tb, s.Commit(tb.Context(), h), tx.ErrTxClosed,
						"and a commit after a rollback is the second terminal operation")
				},
			},
			{
				Method: "Begin",
				Name:   "settles-two-transactions-independently",
				Claim:  "Begin settles two open transactions independently",
				Run: func(tb testing.TB, s tx.Contract, fx txtest.ContractFixture) {
					tb.Helper()
					first, err := s.Begin(tb.Context())
					testkit.NoError(tb, err, "the first transaction opens")
					second, err := s.Begin(tb.Context())
					testkit.NoError(tb, err, "and a second opens beside it")

					testkit.NoError(tb, s.Commit(tb.Context(), first), "the first commits")
					testkit.NoError(tb, s.Rollback(tb.Context(), second),
						"and the second still rolls back — settling one settles one")
				},
			},
			{
				Method: "Begin",
				Name:   "refuses-an-invented-handle",
				Claim:  "Begin refuses a handle that never began",
				Run: func(tb testing.TB, s tx.Contract, fx txtest.ContractFixture) {
					tb.Helper()
					testkit.ErrorIs(tb, s.Commit(tb.Context(), tx.Tx{ID: 99}), tx.ErrTxClosed,
						"there is nothing to commit under an invented handle")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	txtest.RunContract(t,
		txtest.ContractHarness[*txtest.InMemory]{Name: "in-memory", New: txtest.NewInMemory},
		txtest.ContractSuite.Without(txtest.ContractSuite.Checks.Begin.Smoke()),
	)
}

// Staging is on the interface now, and the generated law drives it — but its
// refusals still belong here: a settled handle stages nothing, which is a
// claim about this subject's own error rather than about the contract.
func TestStagingRefusesASettledHandle(t *testing.T) {
	t.Parallel()

	s := txtest.NewInMemory()
	h, err := s.Begin(t.Context())
	testkit.NoError(t, err, "a transaction opens")
	testkit.NoError(t, s.Put(t.Context(), h, "k", tx.Value{Key: "k", Body: "staged"}),
		"the open transaction stages")

	_, err = s.Get(t.Context(), "k")
	testkit.ErrorIs(t, err, tx.ErrNotFound, "and the outside read sees nothing of it")

	testkit.NoError(t, s.Rollback(t.Context(), h), "the transaction rolls back")
	testkit.ErrorIs(t, s.Put(t.Context(), h, "k", tx.Value{Key: "k", Body: "late"}), tx.ErrTxClosed,
		"a settled handle stages nothing")

	h2, err := s.Begin(t.Context())
	testkit.NoError(t, err, "a second transaction opens")
	testkit.NoError(t, s.Put(t.Context(), h2, "k", tx.Value{Key: "k", Body: "kept"}), "and stages")
	testkit.NoError(t, s.Commit(t.Context(), h2), "and commits")

	got, err := s.Get(t.Context(), "k")
	testkit.NoError(t, err, "the committed write is readable")
	testkit.Equal(t, got.Body, "kept", "whole, as staged")
}
