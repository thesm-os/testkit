// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package txtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx/txtest"
	"go.thesmos.sh/testkit/engine/model"
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
		// The mid-transaction write is derived now: Put is on the interface,
		// so the law reaches it the way a consumer would and a defect worn on
		// the interface reaches the law back.
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

// The saturation prover: every bound law must be able to fail as itself,
// with nothing to arm: the staging write is derived from the interface.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	txtest.ContractModelSaturation(t, func() tx.Contract {
		return txtest.NewInMemory()
	})
}

// commitRefuses is the defect only the composite driving can see: every
// Commit claims its handle is already settled. The laws tolerate it — a
// commit-after-rollback answering the closed sentinel is exactly what the
// mutex law wants, and a refused first commit holds the or-rollback law
// vacuously — and the old standalone writers drove the terminals with
// handles no Begin minted, where both sides refusing bogus handles read as
// agreement. Only a driving that threads one Begin's own handle into its
// own commit diverges from the reference here.
type commitRefuses struct{ tx.Contract }

func (commitRefuses) Commit(context.Context, tx.Tx) error {
	return tx.ErrTxClosed
}

// The composite threads one Begin's handle into its own terminal, so a
// commit that refuses what its begin minted diverges from the reference at
// the step's own name.
func TestTwoPhaseCompositeCatchesARefusedOwnHandle(t *testing.T) {
	t.Parallel()

	got := testkit.Rejects(t, "a commit refusing the handle its own begin minted",
		func(tb testing.TB) {
			tb.Helper()
			// The clean factory stands as reference — the twin would wear
			// the same defect and agree, which is the saturation prover's
			// own reason for this option.
			model.Check(tb, txtest.ContractModelProperty(func() tx.Contract {
				return commitRefuses{txtest.NewInMemory()}
			}, txtest.ContractModelReference(func() tx.Contract {
				return txtest.NewInMemory()
			})))
		})
	testkit.Assert(t, got).Contains("Commit: SUT err=",
		"and rejects it at the terminal step the composite drove")
}
