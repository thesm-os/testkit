// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package serializable is the mixin-axis fixture for the serializable mixin,
// which declares that concurrent transactions are equivalent to some serial
// order — so no anomaly is observable at all.
//
// The sibling of [snapshotisolation], and the fixture pair states why they
// are siblings rather than levels. Snapshot isolation forbids dirty writes
// and dirty, intermediate and circular reads, and *permits* write skew: two
// transactions each reading what the other is about to invalidate, both
// committing. Serializability forbids that too. So the anti-dependency-cycle
// check is correct against this claim and wrong against a snapshot, and a
// store that declares only the snapshot must not earn it.
//
// The interface reports its own transaction history, because that is the only
// thing an isolation claim can be checked against: the anomalies are patterns
// across transactions, not properties of any one call.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package serializable

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
//testkit:out serializabletest/ pkg=serializabletest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Record appends one operation to the history.
	Record(ctx context.Context, e Entry) error

	// History reports what the transactions did, which is what the anomaly
	// checks read.
	//testkit:mixin serializable
	History(ctx context.Context) ([]Entry, error)

	// Get reads a key's latest recorded entry — the read half a serializable
	// store genuinely has, and the draw that gives the anomaly law a key type
	// to instantiate at.
	Get(ctx context.Context, key string) (Entry, error)
}
