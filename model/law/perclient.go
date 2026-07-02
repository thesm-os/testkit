// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"fmt"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/trace"
)

// ClientOp is a single read or write extracted from a trace event
// for per-client consistency checking. Version is the store-assigned
// monotonic stamp of the value written or read — the ordering oracle
// the per-client guarantees are defined against (a logical clock, a
// row version, or the global write order). Higher means newer.
type ClientOp[K comparable] struct {
	Write   bool
	Key     K
	Version int64
}

// ClientClassifier maps a trace event to a [ClientOp]. ok=false
// skips the event (not a read or write relevant to the guarantee).
// The generator emits a classifier per interface from the method
// shapes; consumers can supply one directly.
type ClientClassifier[K comparable] func(trace.Event) (op ClientOp[K], ok bool)

// clientKey pairs a client ID with a key for per-(client,key) state
// in the per-client laws. K is comparable so the pair is a valid
// map key.
type clientKey[K comparable] struct {
	client int
	key    K
}

// MonotonicReads verifies that, within a single client and key, the
// versions of successive reads never decrease — once a client reads
// a value, it never later reads an older one. Auto-emitted for the
// //testkit:monotonic-reads directive.
//
// Reads are correlated to the bound per-iteration trace via
// [TraceBinder]; Classify interprets each event.
type MonotonicReads[T any, K comparable] struct {
	Classify ClientClassifier[K]
	Trace    *trace.Trace
}

// BindTrace sets the trace reference; called by the runner at
// iteration start.
func (l *MonotonicReads[T, K]) BindTrace(t *trace.Trace) { l.Trace = t }

// ID returns the stable identifier for this law.
func (*MonotonicReads[T, K]) ID() string { return "AUTO-MONOTONIC-READS" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (*MonotonicReads[T, K]) REQID() string { return "" }

// Check scans the trace and fails if any client reads a strictly
// older version of a key than it read earlier.
func (l *MonotonicReads[T, K]) Check(_ *rapid.T, _, _ T) error {
	last := make(map[clientKey[K]]int64)
	for _, ev := range l.Trace.Snapshot() {
		op, ok := l.Classify(ev)
		if !ok || op.Write {
			continue
		}
		ck := clientKey[K]{client: ev.ClientID, key: op.Key}
		if prev, seen := last[ck]; seen && op.Version < prev {
			return fmt.Errorf("MonotonicReads: client %d key %v read version %d after %d (read went backwards)",
				ev.ClientID, op.Key, op.Version, prev)
		}
		if op.Version > last[ck] {
			last[ck] = op.Version
		}
	}
	return nil
}

// ReadYourWrites verifies that, within a single client and key, a
// read taken after the client's own write returns a version no
// older than that write — the client always observes its own
// effects (or newer ones from other clients). Auto-emitted for the
// //testkit:read-your-writes directive.
type ReadYourWrites[T any, K comparable] struct {
	Classify ClientClassifier[K]
	Trace    *trace.Trace
}

// BindTrace sets the trace reference; called by the runner at
// iteration start.
func (l *ReadYourWrites[T, K]) BindTrace(t *trace.Trace) { l.Trace = t }

// ID returns the stable identifier for this law.
func (*ReadYourWrites[T, K]) ID() string { return "AUTO-READ-YOUR-WRITES" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (*ReadYourWrites[T, K]) REQID() string { return "" }

// Check scans the trace and fails if any client reads a version of
// a key older than the version it most recently wrote to that key.
func (l *ReadYourWrites[T, K]) Check(_ *rapid.T, _, _ T) error {
	ownWrite := make(map[clientKey[K]]int64)
	for _, ev := range l.Trace.Snapshot() {
		op, ok := l.Classify(ev)
		if !ok {
			continue
		}
		ck := clientKey[K]{client: ev.ClientID, key: op.Key}
		if op.Write {
			ownWrite[ck] = op.Version
			continue
		}
		if w, seen := ownWrite[ck]; seen && op.Version < w {
			return fmt.Errorf(
				"ReadYourWrites: client %d key %v read version %d after writing %d (did not observe own write)",
				ev.ClientID,
				op.Key,
				op.Version,
				w,
			)
		}
	}
	return nil
}

// MonotonicWrites verifies that, within a single client and key, the
// versions assigned to the client's successive writes strictly
// increase in issue order — the store serializes a client's writes
// in the order it issued them. Auto-emitted for the
// //testkit:monotonic-writes directive.
type MonotonicWrites[T any, K comparable] struct {
	Classify ClientClassifier[K]
	Trace    *trace.Trace
}

// BindTrace sets the trace reference; called by the runner at
// iteration start.
func (l *MonotonicWrites[T, K]) BindTrace(t *trace.Trace) { l.Trace = t }

// ID returns the stable identifier for this law.
func (*MonotonicWrites[T, K]) ID() string { return "AUTO-MONOTONIC-WRITES" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (*MonotonicWrites[T, K]) REQID() string { return "" }

// Check scans the trace and fails if any client's write to a key is
// stamped with a version not strictly greater than its previous
// write to that key.
func (l *MonotonicWrites[T, K]) Check(_ *rapid.T, _, _ T) error {
	lastWrite := make(map[clientKey[K]]int64)
	for _, ev := range l.Trace.Snapshot() {
		op, ok := l.Classify(ev)
		if !ok || !op.Write {
			continue
		}
		ck := clientKey[K]{client: ev.ClientID, key: op.Key}
		if prev, seen := lastWrite[ck]; seen && op.Version <= prev {
			return fmt.Errorf("MonotonicWrites: client %d key %v wrote version %d after %d (writes not monotonic)",
				ev.ClientID, op.Key, op.Version, prev)
		}
		lastWrite[ck] = op.Version
	}
	return nil
}

// WritesFollowReads verifies that, within a single client, every
// write is stamped no older than any version the client has read so
// far — a write causally follows the reads that preceded it.
// Auto-emitted for the //testkit:writes-follow-reads directive.
//
// The check is conservative and key-agnostic: it tracks the maximum
// version a client has read across all keys and fails a write whose
// version is strictly below it.
type WritesFollowReads[T any, K comparable] struct {
	Classify ClientClassifier[K]
	Trace    *trace.Trace
}

// BindTrace sets the trace reference; called by the runner at
// iteration start.
func (l *WritesFollowReads[T, K]) BindTrace(t *trace.Trace) { l.Trace = t }

// ID returns the stable identifier for this law.
func (*WritesFollowReads[T, K]) ID() string { return "AUTO-WRITES-FOLLOW-READS" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (*WritesFollowReads[T, K]) REQID() string { return "" }

// Check scans the trace and fails if any client issues a write with
// a version older than the newest version it has read.
func (l *WritesFollowReads[T, K]) Check(_ *rapid.T, _, _ T) error {
	maxRead := make(map[int]int64)
	seen := make(map[int]bool)
	for _, ev := range l.Trace.Snapshot() {
		op, ok := l.Classify(ev)
		if !ok {
			continue
		}
		c := ev.ClientID
		if !op.Write {
			if !seen[c] || op.Version > maxRead[c] {
				maxRead[c] = op.Version
				seen[c] = true
			}
			continue
		}
		if seen[c] && op.Version < maxRead[c] {
			return fmt.Errorf(
				"WritesFollowReads: client %d wrote version %d after reading %d (write does not follow read)",
				c,
				op.Version,
				maxRead[c],
			)
		}
	}
	return nil
}
