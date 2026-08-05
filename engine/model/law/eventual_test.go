// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"slices"
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
