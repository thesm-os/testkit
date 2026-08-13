// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/equivalence"
	"go.thesmos.sh/testkit/core/lawid"
)

// CheckEventualConvergence verifies the convergence contract over
// replica state snapshots: every post-sync state must equal the
// merge-join of the pre-sync states. Comparing against the join —
// rather than only pairwise across replicas — rejects a sync that
// "converges" by losing writes (all replicas agreeing on less than
// the union of what they had is still a violation).
//
// equal may be nil, in which case states are compared with cmp.Diff
// (which also produces the diagnostic diff). merge must be the join
// of the replica state lattice: commutative, associative,
// idempotent. The [linearize.Eventual] consistency model delegates
// here.
func CheckEventualConvergence[S any](pre, post []S, merge func(a, b S) S, equal func(a, b S) bool) error {
	if len(pre) != len(post) {
		return fmt.Errorf("eventual law: %d pre-sync states vs %d post-sync states", len(pre), len(post))
	}
	if len(pre) == 0 {
		return nil
	}
	expected := pre[0]
	for _, s := range pre[1:] {
		expected = merge(expected, s)
	}
	for i, s := range post {
		if equal != nil {
			if !equal(s, expected) {
				return fmt.Errorf("eventual law: replica %d did not converge to the join of pre-sync states", i)
			}
			continue
		}
		if diff := cmp.Diff(expected, s); diff != "" {
			return fmt.Errorf(
				"eventual law: replica %d did not converge to the join of pre-sync states (-join +replica):\n%s",
				i,
				diff,
			)
		}
	}
	return nil
}

// EventualConvergence verifies that N replicas receiving disjoint
// slices of a write stream converge — after the quiet window and an
// anti-entropy round — to the join of their pre-sync states.
// Auto-emitted for the //testkit:eventual <window> directive.
//
// The law constructs Replicas fresh instances (default 3), routes
// each drawn value to one of them (rapid picks the split), runs
// Settle (the quiet-window action: advance the injected clock,
// flush delivery buffers; nil to skip), snapshots every replica,
// runs Sync (the SUT's anti-entropy), snapshots again, and
// delegates to [CheckEventualConvergence].
type EventualConvergence[T any, V any, S any] struct {
	Factory func() T

	// Replicas is how many instances receive the write stream. Zero defaults
	// to 3: two replicas cannot exhibit a merge that is order-dependent.
	Replicas int

	Write    func(*rapid.T, T, V) error
	Values   *rapid.Generator[V]
	Settle   func(rt *rapid.T, replicas []T)
	Sync     func(rt *rapid.T, replicas []T) error
	Snapshot func(*rapid.T, T) S
	Merge    func(a, b S) S

	// Equal is the equivalence converged snapshots are held to. Nil is
	// strict deep equality, which is what a generated binding leaves it as;
	// supply a chain where a snapshot carries per-replica bookkeeping — a
	// node identifier, a vector clock — that convergence does not erase.
	Equal *equivalence.Chain
}

// ID returns the stable identifier for this law.
func (EventualConvergence[T, V, S]) ID() string { return lawid.EventualConvergence }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (EventualConvergence[T, V, S]) REQID() string { return "" }

// Check scatters writes across fresh replicas, settles, syncs, and
// verifies convergence to the pre-sync join.
func (l EventualConvergence[T, V, S]) Check(rt *rapid.T, _, _ T) error {
	n := l.Replicas
	if n <= 0 {
		n = 3
	}
	replicas := make([]T, n)
	for i := range n {
		replicas[i] = l.Factory()
	}
	writes := rapid.IntRange(1, 6).Draw(rt, "EventualConvergence_writes")
	for i := range writes {
		v := l.Values.Draw(rt, fmt.Sprintf("EventualConvergence_v%d", i))
		dst := rapid.IntRange(0, n-1).Draw(rt, fmt.Sprintf("EventualConvergence_dst%d", i))
		if err := l.Write(rt, replicas[dst], v); err != nil {
			return Vacuous // a precondition this run supplies was refused
		}
	}
	if l.Settle != nil {
		l.Settle(rt, replicas)
	}
	pre := make([]S, n)
	for i, r := range replicas {
		pre[i] = l.Snapshot(rt, r)
	}
	if err := l.Sync(rt, replicas); err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	post := make([]S, n)
	for i, r := range replicas {
		post[i] = l.Snapshot(rt, r)
	}
	// Adapted rather than passed through: CheckEventualConvergence is
	// exported and takes a typed predicate, which is the right surface for a
	// caller driving it by hand. The Chain is the generated binding's side of
	// the same question.
	//
	// Left nil where no chain was supplied, rather than adapted to one that
	// compares strictly. Both answer the same question, but the nil arm
	// reports the diff that says which field diverged, and a converged-state
	// failure with no diff is the hardest kind to act on.
	var equal func(a, b S) bool
	if l.Equal != nil {
		equal = func(a, b S) bool { return l.Equal.Equal(a, b) }
	}
	return CheckEventualConvergence(pre, post, l.Merge, equal)
}
