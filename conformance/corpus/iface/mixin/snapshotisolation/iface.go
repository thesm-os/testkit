// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package snapshotisolation is the mixin-axis fixture for the
// snapshotisolation mixin, which declares that concurrent transactions are
// free of the read and write anomalies a serial order would prevent.
//
// The interface reports its own transaction history, because that is the only
// thing an isolation claim can be checked against: the anomalies are patterns
// across transactions, not properties of any one call.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package snapshotisolation

import (
	"context"
)

// Entry is one recorded operation, with the transaction that made it.
type Entry struct {
	Txn     int
	Key     string
	Version int64
	Write   bool
}

// Mixed is the fixture interface.
//
//testkit:out snapshotisolationtest/ pkg=snapshotisolationtest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Record appends one operation to the history.
	Record(ctx context.Context, e Entry) error

	// History reports what the transactions did, which is what the anomaly
	// checks read.
	//testkit:mixin snapshotisolation
	History(ctx context.Context) ([]Entry, error)
}
