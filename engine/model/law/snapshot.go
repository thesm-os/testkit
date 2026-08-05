// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"fmt"
	"slices"

	"pgregory.net/rapid"
)

// TxnOp is one read or write observation inside a transaction: the
// key touched and the version read or produced. Versions are the
// store-assigned per-key ordering oracle (a read of version 0 with
// no matching writer denotes the initial state).
type TxnOp[K comparable] struct {
	Key     K
	Version int64
}

// Txn is one transaction in a recorded history, the input to the
// snapshot-isolation anomaly checkers. Reads carry the version each
// read returned (identifying the transaction whose write was
// observed); Writes carry the versions produced. A transaction
// writing the same key more than once lists every write; the
// highest version is its final write, the rest are intermediates.
type Txn[K comparable] struct {
	ID      int
	Aborted bool
	Reads   []TxnOp[K]
	Writes  []TxnOp[K]
}

// siEdgeKind labels a dependency edge in the transaction graph.
type siEdgeKind int

const (
	siWW siEdgeKind = iota // write-dependency: version order on a key
	siWR                   // read-dependency: reader observed writer's final write
	siRW                   // anti-dependency: reader missed a later write
)

// siEdge is a directed dependency edge between txn indices.
type siEdge struct {
	to   int
	kind siEdgeKind
}

// siGraph is the direct-serialization graph over committed
// transactions plus the bookkeeping the G1a/G1b item checks need.
type siGraph[K comparable] struct {
	txns          []Txn[K]
	committed     []int // indices of committed txns
	adj           [][]siEdge
	abortedWrites map[TxnOp[K]]int // (key, version) → aborted txn ID
	intermediates map[TxnOp[K]]int // non-final (key, version) → committed txn index
}

// buildSIGraph constructs the dependency graph per Adya's direct
// serialization graph: ww edges follow per-key version order over
// final writes, wr edges connect a final write to its readers, rw
// edges connect a reader to transactions installing later versions
// of what it read. Aborted transactions contribute no edges; their
// writes are tracked separately for the G1a check.
func buildSIGraph[K comparable](txns []Txn[K]) *siGraph[K] {
	g := &siGraph[K]{
		txns:          txns,
		adj:           make([][]siEdge, len(txns)),
		abortedWrites: make(map[TxnOp[K]]int),
		intermediates: make(map[TxnOp[K]]int),
	}
	// Final write per (txn, key) = highest version; the rest are
	// intermediates. Aborted writes recorded wholesale.
	type verIdx struct {
		version int64
		txn     int
	}
	finalsByKey := make(map[K][]verIdx)
	writerOfFinal := make(map[TxnOp[K]]int)
	for i, t := range txns {
		if t.Aborted {
			for _, w := range t.Writes {
				g.abortedWrites[w] = t.ID
			}
			continue
		}
		g.committed = append(g.committed, i)
		finals := make(map[K]int64)
		for _, w := range t.Writes {
			if w.Version > finals[w.Key] {
				finals[w.Key] = w.Version
			}
		}
		for _, w := range t.Writes {
			if w.Version != finals[w.Key] {
				g.intermediates[w] = i
			}
		}
		for k, v := range finals {
			op := TxnOp[K]{Key: k, Version: v}
			writerOfFinal[op] = i
			finalsByKey[k] = append(finalsByKey[k], verIdx{version: v, txn: i})
		}
	}
	// ww: per-key version order over committed final writes.
	for _, writers := range finalsByKey {
		slices.SortFunc(writers, func(a, b verIdx) int {
			switch {
			case a.version < b.version:
				return -1
			case a.version > b.version:
				return 1
			default:
				return 0
			}
		})
		for i := range writers {
			for j := i + 1; j < len(writers); j++ {
				if writers[i].txn != writers[j].txn {
					g.adj[writers[i].txn] = append(g.adj[writers[i].txn], siEdge{to: writers[j].txn, kind: siWW})
				}
			}
		}
	}
	// wr and rw from committed readers.
	for _, j := range g.committed {
		for _, r := range g.txns[j].Reads {
			if i, ok := writerOfFinal[r]; ok && i != j {
				g.adj[i] = append(g.adj[i], siEdge{to: j, kind: siWR})
			}
			for _, w := range finalsByKey[r.Key] {
				if w.version > r.Version && w.txn != j {
					g.adj[j] = append(g.adj[j], siEdge{to: w.txn, kind: siRW})
				}
			}
		}
	}
	return g
}

// findCycle returns the txn IDs on a cycle reachable through edges
// whose kind satisfies allowed, or nil when the filtered graph is
// acyclic.
func (g *siGraph[K]) findCycle(allowed func(siEdgeKind) bool) []int {
	const (
		white, gray, black = 0, 1, 2
	)
	color := make([]int, len(g.txns))
	var stack []int
	var cycle []int
	var visit func(u int) bool
	visit = func(u int) bool {
		color[u] = gray
		stack = append(stack, u)
		for _, e := range g.adj[u] {
			if !allowed(e.kind) {
				continue
			}
			if color[e.to] == gray {
				// Extract the cycle from the stack.
				at := slices.Index(stack, e.to)
				for _, idx := range stack[at:] {
					cycle = append(cycle, g.txns[idx].ID)
				}
				return true
			}
			if color[e.to] == white && visit(e.to) {
				return true
			}
		}
		stack = stack[:len(stack)-1]
		color[u] = black
		return false
	}
	for _, i := range g.committed {
		if color[i] == white && visit(i) {
			return cycle
		}
	}
	return nil
}

// reaches reports whether to is reachable from over all edges.
func (g *siGraph[K]) reaches(from, to int) bool {
	seen := make([]bool, len(g.txns))
	queue := []int{from}
	seen[from] = true
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		if u == to {
			return true
		}
		for _, e := range g.adj[u] {
			if !seen[e.to] {
				seen[e.to] = true
				queue = append(queue, e.to)
			}
		}
	}
	return false
}

// CheckSIG0 reports a G0 (write cycle / dirty write) anomaly: a
// cycle of write-dependencies among committed transactions. The
// [linearize.SnapshotIsolation] consistency model delegates here.
func CheckSIG0[K comparable](txns []Txn[K]) error {
	g := buildSIGraph(txns)
	if cycle := g.findCycle(func(k siEdgeKind) bool { return k == siWW }); cycle != nil {
		return fmt.Errorf("snapshot-isolation law: G0 write cycle among txns %v", cycle)
	}
	return nil
}

// CheckSIG1 reports G1 anomalies: G1a (a committed transaction read
// a version written by an aborted transaction), G1b (a committed
// transaction read another transaction's intermediate write), or
// G1c (a cycle of write- and read-dependencies).
func CheckSIG1[K comparable](txns []Txn[K]) error {
	g := buildSIGraph(txns)
	for _, j := range g.committed {
		for _, r := range g.txns[j].Reads {
			if abortedID, ok := g.abortedWrites[r]; ok {
				return fmt.Errorf(
					"snapshot-isolation law: G1a: txn %d read key %v version %d written by aborted txn %d",
					g.txns[j].ID,
					r.Key,
					r.Version,
					abortedID,
				)
			}
			if i, ok := g.intermediates[r]; ok && i != j {
				return fmt.Errorf("snapshot-isolation law: G1b: txn %d read intermediate write %v@%d of txn %d",
					g.txns[j].ID, r.Key, r.Version, g.txns[i].ID)
			}
		}
	}
	if cycle := g.findCycle(func(k siEdgeKind) bool { return k == siWW || k == siWR }); cycle != nil {
		return fmt.Errorf("snapshot-isolation law: G1c: dependency cycle among txns %v", cycle)
	}
	return nil
}

// CheckSIG2 reports a G2 (anti-dependency cycle) anomaly: a
// dependency cycle containing at least one rw edge — the write-skew
// class of anomalies.
func CheckSIG2[K comparable](txns []Txn[K]) error {
	g := buildSIGraph(txns)
	for u := range g.adj {
		for _, e := range g.adj[u] {
			if e.kind != siRW {
				continue
			}
			if g.reaches(e.to, u) {
				return fmt.Errorf("snapshot-isolation law: G2: anti-dependency cycle through txns %d and %d",
					g.txns[u].ID, g.txns[e.to].ID)
			}
		}
	}
	return nil
}

// SnapshotIsolationG0 verifies the recorded transaction history is
// free of G0 write-cycle anomalies. Auto-emitted for the
// //testkit:snapshot-isolation directive; History extracts the
// per-iteration transaction history from the SUT (or the runner's
// trace).
type SnapshotIsolationG0[T any, K comparable] struct {
	History func(*rapid.T, T) []Txn[K]
}

// ID returns the stable identifier for this law.
func (SnapshotIsolationG0[T, K]) ID() string { return "AUTO-SNAPSHOT-ISOLATION-G0" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (SnapshotIsolationG0[T, K]) REQID() string { return "" }

// Check delegates the extracted history to [CheckSIG0].
func (l SnapshotIsolationG0[T, K]) Check(rt *rapid.T, sut, _ T) error {
	return CheckSIG0(l.History(rt, sut))
}

// SnapshotIsolationG1 verifies the recorded transaction history is
// free of G1 anomalies (aborted reads, intermediate reads, and
// dependency cycles). Auto-emitted for //testkit:snapshot-isolation.
type SnapshotIsolationG1[T any, K comparable] struct {
	History func(*rapid.T, T) []Txn[K]
}

// ID returns the stable identifier for this law.
func (SnapshotIsolationG1[T, K]) ID() string { return "AUTO-SNAPSHOT-ISOLATION-G1" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (SnapshotIsolationG1[T, K]) REQID() string { return "" }

// Check delegates the extracted history to [CheckSIG1].
func (l SnapshotIsolationG1[T, K]) Check(rt *rapid.T, sut, _ T) error {
	return CheckSIG1(l.History(rt, sut))
}

// SnapshotIsolationG2 verifies the recorded transaction history is
// free of G2 anti-dependency-cycle anomalies (write skew).
// Auto-emitted for //testkit:snapshot-isolation.
type SnapshotIsolationG2[T any, K comparable] struct {
	History func(*rapid.T, T) []Txn[K]
}

// ID returns the stable identifier for this law.
func (SnapshotIsolationG2[T, K]) ID() string { return "AUTO-SNAPSHOT-ISOLATION-G2" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (SnapshotIsolationG2[T, K]) REQID() string { return "" }

// Check delegates the extracted history to [CheckSIG2].
func (l SnapshotIsolationG2[T, K]) Check(rt *rapid.T, sut, _ T) error {
	return CheckSIG2(l.History(rt, sut))
}
