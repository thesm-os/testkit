// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package txwithretry stacks the tx contract with the retrysucceeds mixin on
// the commit role.
//
// It earns a composite fixture because retry changes what the contract's
// terminal-state rule means. Tx says a transaction reaches exactly one
// terminal state: once committed it cannot be rolled back, and once rolled
// back it cannot be committed. Retry says a call failing transiently succeeds
// on a later attempt.
//
// Together they ask whether a commit that failed reached a terminal state. If
// it did, the retry is a second terminal operation and the contract rejects
// it; if it did not, the retry is the same operation continuing. The answer
// here is the second — a failed commit leaves the transaction open — and a
// generator that assumes the first emits a suite failing on the retry it was
// told to expect.
package txwithretry

import (
	"context"
	"errors"
)

// ErrClosed reports a terminal operation on a settled transaction.
var ErrClosed = errors.New("txwithretry: transaction already settled")

// ErrTransient reports a commit that may succeed on retry. It is deliberately
// distinct from ErrClosed: the difference between them is the whole question
// this fixture poses.
var ErrTransient = errors.New("txwithretry: transient commit failure")

// TxWithRetry is the fixture interface.
//
//testkit:out txwithretrytest/ pkg=txwithretrytest
//testkit:stub
//testkit:suite
type TxWithRetry interface {
	// Begin hosts the tx contract and names both partners.
	//testkit:contract tx role=begin commit=Commit rollback=Rollback
	Begin(ctx context.Context) error

	// Commit is the tx contract's commit role and retries on transient
	// failure. A transient failure leaves the transaction open, so the retry
	// is the same terminal operation rather than a second one.
	//testkit:mixin retrysucceeds
	Commit(ctx context.Context) error

	// Rollback is the tx contract's rollback role.
	Rollback(ctx context.Context) error
}
