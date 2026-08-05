// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
)

// syncReplica is an eventually-consistent set replica. When buffered
// is set, writes land in a pending buffer that only becomes visible
// after settle() — modelling the quiet window before anti-entropy.
type syncReplica struct {
	buffered bool
	pending  map[string]struct{}
	state    map[string]struct{}
}

func newSyncReplica(buffered bool) *syncReplica {
	return &syncReplica{
		buffered: buffered,
		pending:  map[string]struct{}{},
		state:    map[string]struct{}{},
	}
}

func (r *syncReplica) write(v string) {
	if r.buffered {
		r.pending[v] = struct{}{}
		return
	}
	r.state[v] = struct{}{}
}

func (r *syncReplica) settle() {
	for v := range r.pending {
		r.state[v] = struct{}{}
	}
	r.pending = map[string]struct{}{}
}

func (r *syncReplica) snapshot() []string {
	out := make([]string, 0, len(r.state))
	for v := range r.state {
		out = append(out, v)
	}
	slices.Sort(out)
	return out
}

// unionSync is a correct anti-entropy round: every replica ends up
// with the union of all visible states.
func unionSync(replicas []*syncReplica) {
	union := map[string]struct{}{}
	for _, r := range replicas {
		for v := range r.state {
			union[v] = struct{}{}
		}
	}
	for _, r := range replicas {
		for v := range union {
			r.state[v] = struct{}{}
		}
	}
}

// mergeSorted is the join on snapshot values: sorted set union.
func mergeSorted(a, b []string) []string {
	seen := map[string]struct{}{}
	for _, v := range a {
		seen[v] = struct{}{}
	}
	for _, v := range b {
		seen[v] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	slices.Sort(out)
	return out
}

func TestEventualConvergence(t *testing.T) {
	t.Parallel()

	t.Run("union anti-entropy converges all replicas to the join", func(t *testing.T) {
		t.Parallel()
		l := law.EventualConvergence[*syncReplica, string, []string]{
			Factory:  func() *syncReplica { return newSyncReplica(false) },
			Replicas: 3,
			Write:    func(_ *rapid.T, r *syncReplica, v string) error { r.write(v); return nil },
			Values:   rapid.SampledFrom([]string{"a", "b", "c"}),
			Sync:     func(_ *rapid.T, rs []*syncReplica) error { unionSync(rs); return nil },
			Snapshot: func(_ *rapid.T, r *syncReplica) []string { return r.snapshot() },
			Merge:    mergeSorted,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("settle runs before sync so buffered writes converge", func(t *testing.T) {
		t.Parallel()
		l := law.EventualConvergence[*syncReplica, string, []string]{
			Factory:  func() *syncReplica { return newSyncReplica(true) },
			Replicas: 3,
			Write:    func(_ *rapid.T, r *syncReplica, v string) error { r.write(v); return nil },
			Values:   rapid.SampledFrom([]string{"a", "b", "c"}),
			Settle: func(_ *rapid.T, rs []*syncReplica) {
				for _, r := range rs {
					r.settle()
				}
			},
			Sync:     func(_ *rapid.T, rs []*syncReplica) error { unionSync(rs); return nil },
			Snapshot: func(_ *rapid.T, r *syncReplica) []string { return r.snapshot() },
			Merge:    mergeSorted,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("no-op sync that never propagates is caught", func(t *testing.T) {
		t.Parallel()
		// Unique values: a duplicated value routed to every replica
		// could make a no-op sync converge by accident.
		l := law.EventualConvergence[*syncReplica, string, []string]{
			Factory:  func() *syncReplica { return newSyncReplica(false) },
			Replicas: 3,
			Write:    func(_ *rapid.T, r *syncReplica, v string) error { r.write(v); return nil },
			Values:   uniqueStringGen(),
			Sync:     func(_ *rapid.T, _ []*syncReplica) error { return nil }, // BUG: no propagation
			Snapshot: func(_ *rapid.T, r *syncReplica) []string { return r.snapshot() },
			Merge:    mergeSorted,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err == nil {
				rt.Fatal("expected non-propagating sync to be caught")
			}
		})
	})

	t.Run("lossy sync that drops writes is caught", func(t *testing.T) {
		t.Parallel()
		l := law.EventualConvergence[*syncReplica, string, []string]{
			Factory:  func() *syncReplica { return newSyncReplica(false) },
			Replicas: 3,
			Write:    func(_ *rapid.T, r *syncReplica, v string) error { r.write(v); return nil },
			Values:   uniqueStringGen(),
			Sync: func(_ *rapid.T, rs []*syncReplica) error {
				// BUG: "converges" by resetting everyone to empty —
				// pairwise equal, but the pre-sync writes are lost.
				for _, r := range rs {
					r.state = map[string]struct{}{}
				}
				return nil
			},
			Snapshot: func(_ *rapid.T, r *syncReplica) []string { return r.snapshot() },
			Merge:    mergeSorted,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err == nil {
				rt.Fatal("expected data-losing sync to be caught")
			}
		})
	})
}

// CheckEventualConvergence compares each post-sync replica against the join of
// the pre-sync states. Convergence on *less* than that join is the subtle
// failure: every replica agrees, but writes were lost on the way.
func TestCheckEventualConvergenceBranches(t *testing.T) {
	t.Parallel()

	union := func(a, b []int) []int {
		seen := map[int]bool{}
		var outv []int
		for _, s := range [][]int{a, b} {
			for _, v := range s {
				if !seen[v] {
					seen[v] = true
					outv = append(outv, v)
				}
			}
		}
		slices.Sort(outv)
		return outv
	}

	t.Run("mismatched replica counts are reported", func(t *testing.T) {
		t.Parallel()
		err := law.CheckEventualConvergence([][]int{{1}}, [][]int{{1}, {1}}, union, nil)
		if err == nil {
			t.Fatal("differing pre and post replica counts cannot be compared")
		}
		if !strings.Contains(err.Error(), "pre-sync") {
			t.Fatalf("the diagnostic must name the mismatch, got: %v", err)
		}
	})

	t.Run("no replicas converges trivially", func(t *testing.T) {
		t.Parallel()
		if err := law.CheckEventualConvergence(nil, nil, union, nil); err != nil {
			t.Fatalf("an empty replica set cannot diverge: %v", err)
		}
	})

	t.Run("converging on the join passes", func(t *testing.T) {
		t.Parallel()
		pre := [][]int{{1}, {2}}
		post := [][]int{{1, 2}, {1, 2}}
		if err := law.CheckEventualConvergence(pre, post, union, nil); err != nil {
			t.Fatalf("replicas holding the join have converged: %v", err)
		}
	})

	// Agreeing on a subset of the join is the lost-write case.
	t.Run("converging on less than the join is a violation", func(t *testing.T) {
		t.Parallel()
		pre := [][]int{{1}, {2}}
		post := [][]int{{1}, {1}} // agreed, but 2 was dropped
		err := law.CheckEventualConvergence(pre, post, union, nil)
		if err == nil {
			t.Fatal("agreement that loses a write is not convergence")
		}
		if !strings.Contains(err.Error(), "replica 0") {
			t.Fatalf("the diagnostic must identify the replica, got: %v", err)
		}
	})

	// A supplied equality replaces the diff-based comparison, which changes
	// the diagnostic but not the verdict.
	t.Run("a supplied equality is used instead of cmp", func(t *testing.T) {
		t.Parallel()
		pre := [][]int{{1}, {2}}
		post := [][]int{{9}, {9}}
		lenient := func(a, b []int) bool { return true }
		if err := law.CheckEventualConvergence(pre, post, union, lenient); err != nil {
			t.Fatalf("a permissive equality must accept: %v", err)
		}
		strict := slices.Equal[[]int]
		err := law.CheckEventualConvergence(pre, post, union, strict)
		if err == nil {
			t.Fatal("a strict equality must reject divergent replicas")
		}
		if strings.Contains(err.Error(), "-join +replica") {
			t.Fatalf("a supplied equality skips the cmp diff, got: %v", err)
		}
	})
}

func TestEventualConvergencePreconditions(t *testing.T) {
	t.Parallel()

	boom := errors.New("refused")

	t.Run("a refused replica write is a precondition", func(t *testing.T) {
		t.Parallel()
		l := law.EventualConvergence[*syncReplica, string, []string]{
			Factory:  func() *syncReplica { return newSyncReplica(false) },
			Replicas: 3,
			Write:    func(*rapid.T, *syncReplica, string) error { return boom },
			Values:   rapid.SampledFrom([]string{"a", "b", "c"}),
			Sync:     func(_ *rapid.T, rs []*syncReplica) error { unionSync(rs); return nil },
			Snapshot: func(_ *rapid.T, r *syncReplica) []string { return r.snapshot() },
			Merge:    mergeSorted,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatalf("a refused write is a precondition: %v", err)
			}
		})
	})

	// Anti-entropy that declines to run leaves nothing to converge; the law
	// must not read that as divergence.
	t.Run("a refused Sync is a precondition", func(t *testing.T) {
		t.Parallel()
		l := law.EventualConvergence[*syncReplica, string, []string]{
			Factory:  func() *syncReplica { return newSyncReplica(false) },
			Replicas: 3,
			Write:    func(_ *rapid.T, r *syncReplica, v string) error { r.write(v); return nil },
			Values:   rapid.SampledFrom([]string{"a", "b", "c"}),
			Sync:     func(*rapid.T, []*syncReplica) error { return boom },
			Snapshot: func(_ *rapid.T, r *syncReplica) []string { return r.snapshot() },
			Merge:    mergeSorted,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatalf("a refused Sync is a precondition: %v", err)
			}
		})
	})

	// Replicas defaults to 3 rather than 0, or the law would sync an empty
	// replica set and pass unconditionally.
	t.Run("a non-positive replica count falls back to the default", func(t *testing.T) {
		t.Parallel()
		var widest int
		l := law.EventualConvergence[*syncReplica, string, []string]{
			Factory: func() *syncReplica { return newSyncReplica(false) },
			Write:   func(_ *rapid.T, r *syncReplica, v string) error { r.write(v); return nil },
			Values:  rapid.SampledFrom([]string{"a", "b", "c"}),
			Sync: func(_ *rapid.T, rs []*syncReplica) error {
				widest = max(widest, len(rs))
				unionSync(rs)
				return nil
			},
			Snapshot: func(_ *rapid.T, r *syncReplica) []string { return r.snapshot() },
			Merge:    mergeSorted,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatalf("converging replicas must pass: %v", err)
			}
		})
		if widest != 3 {
			t.Fatalf("Replicas=0 must fall back to 3, saw %d", widest)
		}
	})
}
