// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"fmt"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/trace"
)

// ClientEvent pairs a [ClientOp] with the client that performed it —
// the flattened form of a trace event the consistency checkers scan.
type ClientEvent[K comparable] struct {
	Client int
	Op     ClientOp[K]
}

// CheckCausalOrder verifies a sequence of client observations
// respects causal consistency: once a client has observed a write,
// every read it performs afterwards must reflect the full causal cut
// of that write. Non-linearizable orderings are permitted — clients
// may lag arbitrarily — but no client may see effect without cause.
//
// happensBefore is the causal relation over writes (both arguments
// have Op.Write set). It must be transitively closed; the checker
// does not compute the closure. A client "observes" a write by
// performing it or by reading its (key, version); observation pulls
// every happens-before predecessor into the client's causal cut, and
// a later read of a key below the cut's required version fails.
// Session monotonicity is included: re-reading an older version of a
// previously seen key fails even without a happens-before edge.
//
// The [linearize.Causal] consistency model delegates here.
func CheckCausalOrder[K comparable](events []ClientEvent[K], happensBefore func(a, b ClientOp[K]) bool) error {
	var writes []ClientOp[K]
	for _, ev := range events {
		if ev.Op.Write {
			writes = append(writes, ev.Op)
		}
	}
	required := make(map[int]map[K]int64)
	cut := func(client int) map[K]int64 {
		m, ok := required[client]
		if !ok {
			m = make(map[K]int64)
			required[client] = m
		}
		return m
	}
	observe := func(reqs map[K]int64, op ClientOp[K]) {
		for _, w := range writes {
			if happensBefore(w, op) && w.Version > reqs[w.Key] {
				reqs[w.Key] = w.Version
			}
		}
		if op.Version > reqs[op.Key] {
			reqs[op.Key] = op.Version
		}
	}
	for i, ev := range events {
		reqs := cut(ev.Client)
		if ev.Op.Write {
			observe(reqs, ev.Op)
			continue
		}
		if want := reqs[ev.Op.Key]; ev.Op.Version < want {
			return fmt.Errorf(
				"causal law: event %d: client %d read key %v at version %d but its causal cut requires >= %d",
				i, ev.Client, ev.Op.Key, ev.Op.Version, want)
		}
		// Reading (key, version) observes the write that produced it,
		// pulling that write's causal predecessors into the cut.
		observed := false
		for _, w := range writes {
			if w.Key == ev.Op.Key && w.Version == ev.Op.Version {
				observe(reqs, w)
				observed = true
				break
			}
		}
		if !observed && ev.Op.Version > reqs[ev.Op.Key] {
			reqs[ev.Op.Key] = ev.Op.Version
		}
	}
	return nil
}

// CausalOrdering verifies the per-iteration trace respects causal
// consistency per [CheckCausalOrder]. Auto-emitted for the
// //testkit:causal directive.
//
// Classify maps trace events to read/write observations carrying the
// store-assigned version stamp (see [ClientClassifier]);
// HappensBefore is the transitively-closed causal relation over
// writes.
type CausalOrdering[T any, K comparable] struct {
	Classify      ClientClassifier[K]
	HappensBefore func(a, b ClientOp[K]) bool
	Trace         *trace.Trace
}

// BindTrace sets the trace reference; called by the runner at
// iteration start.
func (l *CausalOrdering[T, K]) BindTrace(t *trace.Trace) { l.Trace = t }

// ID returns the stable identifier for this law.
func (*CausalOrdering[T, K]) ID() string { return "AUTO-CAUSAL-ORDERING" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (*CausalOrdering[T, K]) REQID() string { return "" }

// Check flattens the trace through Classify and delegates to
// [CheckCausalOrder].
func (l *CausalOrdering[T, K]) Check(_ *rapid.T, _, _ T) error {
	var events []ClientEvent[K]
	for _, ev := range l.Trace.Snapshot() {
		op, ok := l.Classify(ev)
		if !ok {
			continue
		}
		events = append(events, ClientEvent[K]{Client: ev.ClientID, Op: op})
	}
	return CheckCausalOrder(events, l.HappensBefore)
}
