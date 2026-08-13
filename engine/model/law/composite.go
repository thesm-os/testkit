// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"errors"
	"fmt"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
)

// PoolBalancedGetPut verifies that the running Get-vs-Put delta
// stays non-negative across the test run and returns to zero at
// quiescence. Auto-emitted for methods carrying //testkit:pool.
//
// The law observes via consumer-supplied Stats which returns the
// (gets, puts, outstanding) triple. Stats may be implemented in
// terms of [refpool.BalancedPool.Stats] when the reference is
// available.
type PoolBalancedGetPut[T any] struct {
	Stats func(*rapid.T, T) (gets, puts, outstanding int)
}

// ID returns the stable identifier for this law.
func (PoolBalancedGetPut[T]) ID() string { return lawid.PoolBalanced }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PoolBalancedGetPut[T]) REQID() string { return "" }

// Check verifies outstanding == gets - puts and outstanding ≥ 0.
func (l PoolBalancedGetPut[T]) Check(rt *rapid.T, sut, _ T) error {
	gets, puts, outstanding := l.Stats(rt, sut)
	if outstanding < 0 {
		return fmt.Errorf("PoolBalancedGetPut: negative outstanding %d (gets=%d puts=%d)", outstanding, gets, puts)
	}
	if outstanding != gets-puts {
		return fmt.Errorf("PoolBalancedGetPut: outstanding %d != gets %d - puts %d", outstanding, gets, puts)
	}
	return nil
}

// PoolLeakFree verifies that at quiescence (no outstanding cycle)
// the pool reports balanced state. Auto-emitted alongside
// PoolBalancedGetPut. This law fires once per iteration at the
// end-of-iteration boundary; the runner calls Check after the
// final action when the consumer's Cycle reports complete.
type PoolLeakFree[T any] struct {
	Balanced func(*rapid.T, T) bool
}

// ID returns the stable identifier for this law.
func (PoolLeakFree[T]) ID() string { return lawid.PoolLeakFree }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PoolLeakFree[T]) REQID() string { return "" }

// Check verifies the pool reports Balanced=true.
func (l PoolLeakFree[T]) Check(rt *rapid.T, sut, _ T) error {
	if !l.Balanced(rt, sut) {
		return errors.New("law: pool-leak-free: pool reports outstanding resources at quiescence")
	}
	return nil
}

// CursorCloseIdempotent verifies a second Close on the cursor is
// a no-op (returns nil and does not error). Auto-emitted for
// methods carrying //testkit:cursor <Close>.
type CursorCloseIdempotent[T any] struct {
	Close func(*rapid.T, T) error
}

// IsolatedLaw marks the conduct: this Check corrupts its subjects to make
// its observation, and the runner hands it a throwaway pair of its own.
func (CursorCloseIdempotent[T]) IsolatedLaw() {}

// ID returns the stable identifier for this law.
func (CursorCloseIdempotent[T]) ID() string { return lawid.CursorCloseIdempotent }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (CursorCloseIdempotent[T]) REQID() string { return "" }

// Check closes twice and verifies the second close is silent.
func (l CursorCloseIdempotent[T]) Check(rt *rapid.T, sut, _ T) error {
	_ = l.Close(rt, sut)
	if err := l.Close(rt, sut); err != nil {
		return fmt.Errorf("CursorCloseIdempotent: second close errored: %v", err)
	}
	return nil
}

// CursorNextAfterCloseSentinel verifies that calling Next after
// Close returns the configured sentinel error.
type CursorNextAfterCloseSentinel[T any, V any] struct {
	Close    func(*rapid.T, T) error
	Next     func(*rapid.T, T) (V, bool, error)
	Sentinel error
}

// IsolatedLaw marks the conduct: this Check corrupts its subjects to make
// its observation, and the runner hands it a throwaway pair of its own.
func (CursorNextAfterCloseSentinel[T, V]) IsolatedLaw() {}

// ID returns the stable identifier for this law.
func (CursorNextAfterCloseSentinel[T, V]) ID() string { return lawid.CursorNextAfterClose }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (CursorNextAfterCloseSentinel[T, V]) REQID() string { return "" }

// Check closes and then asserts Next returns the sentinel.
func (l CursorNextAfterCloseSentinel[T, V]) Check(rt *rapid.T, sut, _ T) error {
	if err := l.Close(rt, sut); err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	_, _, err := l.Next(rt, sut)
	if !errors.Is(err, l.Sentinel) {
		return fmt.Errorf("CursorNextAfterCloseSentinel: Next after Close returned %v (want %v)", err, l.Sentinel)
	}
	return nil
}

// TwoPhaseNoRollbackAfterCommit verifies that calling Rollback on
// a transaction that has already been committed returns the
// configured tx-closed error. Auto-emitted for methods carrying
// //testkit:two-phase <Commit> <Rollback>.
type TwoPhaseNoRollbackAfterCommit[T any, Tx any] struct {
	Begin    func(*rapid.T, T) (Tx, error)
	Commit   func(*rapid.T, T, Tx) error
	Rollback func(*rapid.T, T, Tx) error
	Closed   error
}

// ID returns the stable identifier for this law.
func (TwoPhaseNoRollbackAfterCommit[T, Tx]) ID() string {
	return lawid.TwoPhaseRollbackAfterCommit
}

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (TwoPhaseNoRollbackAfterCommit[T, Tx]) REQID() string { return "" }

// Check begins, commits, then asserts a follow-up rollback returns
// the closed sentinel.
func (l TwoPhaseNoRollbackAfterCommit[T, Tx]) Check(rt *rapid.T, sut, _ T) error {
	tx, beginErr := l.Begin(rt, sut)
	if beginErr != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	if commitErr := l.Commit(rt, sut, tx); commitErr != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	err := l.Rollback(rt, sut, tx)
	if !errors.Is(err, l.Closed) {
		return fmt.Errorf("two-phase law: rollback after commit returned %v (want %v)", err, l.Closed)
	}
	return nil
}

// TwoPhaseCommitOrRollback verifies the commit-XOR-rollback mutex:
// once a transaction reaches a terminal state via Commit or
// Rollback, the other terminal operation must fail with the closed
// sentinel. Auto-emitted for methods carrying
// //testkit:two-phase <Commit> <Rollback>.
//
// rapid draws which terminal operation runs first, so both
// orderings — commit-then-rollback and rollback-then-commit — are
// exercised. The first terminal operation must succeed; the second
// must return Closed. (The narrower [TwoPhaseNoRollbackAfterCommit]
// covers only the commit-then-rollback direction.)
type TwoPhaseCommitOrRollback[T any, Tx any] struct {
	Begin    func(*rapid.T, T) (Tx, error)
	Commit   func(*rapid.T, T, Tx) error
	Rollback func(*rapid.T, T, Tx) error
	Closed   error
}

// ID returns the stable identifier for this law.
func (TwoPhaseCommitOrRollback[T, Tx]) ID() string { return lawid.TwoPhaseMutex }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (TwoPhaseCommitOrRollback[T, Tx]) REQID() string { return "" }

// Check begins a transaction, runs one terminal operation (drawn by
// rapid), and asserts the other terminal operation is then rejected
// with the closed sentinel.
func (l TwoPhaseCommitOrRollback[T, Tx]) Check(rt *rapid.T, sut, _ T) error {
	tx, beginErr := l.Begin(rt, sut)
	if beginErr != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	first, second, firstName, secondName := l.Commit, l.Rollback, "commit", "rollback"
	if rapid.Bool().Draw(rt, "TwoPhaseCommitOrRollback_rollbackFirst") {
		first, second, firstName, secondName = l.Rollback, l.Commit, "rollback", "commit"
	}
	if err := first(rt, sut, tx); err != nil {
		return nil //nolint:nilerr // first terminal op failed; law vacuously holds
	}
	err := second(rt, sut, tx)
	if !errors.Is(err, l.Closed) {
		return fmt.Errorf("TwoPhaseCommitOrRollback: %s after %s returned %v (want closed=%v)",
			secondName, firstName, err, l.Closed)
	}
	return nil
}

// SagaFullCompensation verifies that when a saga step fails the
// observable state at the end of Run equals the pre-Run snapshot
// (compensation undid every prior committed step). Auto-emitted
// for methods carrying //testkit:saga.
type SagaFullCompensation[T any, Obs comparable] struct {
	Run     func(*rapid.T, T) error
	Observe func(*rapid.T, T) Obs
}

// ID returns the stable identifier for this law.
func (SagaFullCompensation[T, Obs]) ID() string { return lawid.SagaFullCompensation }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (SagaFullCompensation[T, Obs]) REQID() string { return "" }

// Check observes before and after Run; when Run errored the
// post-state must equal the pre-state.
func (l SagaFullCompensation[T, Obs]) Check(rt *rapid.T, sut, ref T) error {
	before := l.Observe(rt, sut)
	if err := l.Run(rt, sut); err == nil {
		// A completed saga mutated the subject; the same run lands on the
		// reference — the mirrored half of the [Law] conduct contract. A
		// failed run compensated itself, and mirrors nothing.
		return mirror("SagaFullCompensation", func() error { return l.Run(rt, ref) })
	}
	after := l.Observe(rt, sut)
	if before != after {
		return fmt.Errorf("SagaFullCompensation: failure left state mutated: before=%v after=%v", before, after)
	}
	return nil
}
