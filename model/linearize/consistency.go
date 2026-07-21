// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"go.thesmos.sh/testkit/model/law"
)

// The consistency models below are the runner-facing dispatch points
// for the non-linearizable consistency selectors (//testkit:eventual,
// //testkit:causal, //testkit:snapshot-isolation). Unlike the
// Porcupine builders in this package they do not produce a
// porcupine.Model — their checking algorithms operate on replica
// snapshots, client observations, and transaction histories, and
// live in model/law (which this package may import; the reverse
// would cycle through model). Each type here is the named model the
// runner or an emitted harness selects; the corresponding law
// (law.EventualConvergence, law.CausalOrdering,
// law.SnapshotIsolationG0/G1/G2) is the per-iteration registry unit.

// Eventual is the eventual-consistency model: replicas must
// converge, after the quiet window and anti-entropy, to the join of
// their pre-sync states under Merge. Equal may be nil to compare
// with go-cmp. Delegates to [law.CheckEventualConvergence].
type Eventual[S any] struct {
	Merge func(a, b S) S
	Equal func(a, b S) bool
}

// Check verifies every post-sync state equals the join of the
// pre-sync states.
func (e Eventual[S]) Check(pre, post []S) error {
	return law.CheckEventualConvergence(pre, post, e.Merge, e.Equal)
}

// Causal is the causal-consistency model: clients may lag behind
// the newest state, but no client observes an effect without its
// causes. HappensBefore is the transitively-closed causal relation
// over writes. Delegates to [law.CheckCausalOrder].
type Causal[K comparable] struct {
	HappensBefore func(a, b law.ClientOp[K]) bool
}

// Check verifies the observation sequence respects the causal cuts
// implied by HappensBefore.
func (c Causal[K]) Check(events []law.ClientEvent[K]) error {
	return law.CheckCausalOrder(events, c.HappensBefore)
}

// SnapshotIsolation is the snapshot-isolation model: transaction
// histories are checked for the G0 (write cycle), G1 (aborted read,
// intermediate read, dependency cycle), and G2 (anti-dependency
// cycle) anomaly classes. Delegates to [law.CheckSIG0],
// [law.CheckSIG1], and [law.CheckSIG2].
type SnapshotIsolation[K comparable] struct{}

// G0 reports write-cycle anomalies in the history.
func (SnapshotIsolation[K]) G0(txns []law.Txn[K]) error { return law.CheckSIG0(txns) }

// G1 reports aborted-read, intermediate-read, and dependency-cycle
// anomalies in the history.
func (SnapshotIsolation[K]) G1(txns []law.Txn[K]) error { return law.CheckSIG1(txns) }

// G2 reports anti-dependency-cycle (write skew) anomalies in the
// history.
func (SnapshotIsolation[K]) G2(txns []law.Txn[K]) error { return law.CheckSIG2(txns) }
