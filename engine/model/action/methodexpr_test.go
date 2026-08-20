// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"slices"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/action"
)

// mxIface is the surface the method expressions are taken from — the
// constructors' whole claim is about interface method expressions, so
// the test drives them through one.
type mxIface interface {
	Put(ctx context.Context, v string) error
	Get(ctx context.Context, k string) (string, error)
	Len(ctx context.Context) (int, error)
	Drop(ctx context.Context, k string) error
	Close(ctx context.Context) error
	Peek(ctx context.Context, k string) (string, bool)
	Set(ctx context.Context, k, v string) error
	Acquire(ctx context.Context) (int, error)
	Release(ctx context.Context, r int) error
	All(ctx context.Context) ([]string, error)
}

// mxProbe answers benignly and records every call with its arguments,
// so the test can assert the constructor invoked the method it names —
// the defect class these constructors exist to remove.
type mxProbe struct {
	calls *[]string
	kv    map[string]string
}

func (p mxProbe) note(s string) { *p.calls = append(*p.calls, s) }

func (p mxProbe) Put(_ context.Context, v string) error { p.note("Put:" + v); return nil }
func (p mxProbe) Get(_ context.Context, k string) (string, error) {
	p.note("Get:" + k)
	return p.kv[k], nil
}
func (p mxProbe) Len(context.Context) (int, error)       { p.note("Len"); return len(p.kv), nil }
func (p mxProbe) Drop(_ context.Context, k string) error { p.note("Drop:" + k); return nil }
func (p mxProbe) Close(context.Context) error            { p.note("Close"); return nil }
func (p mxProbe) Peek(_ context.Context, k string) (string, bool) {
	p.note("Peek:" + k)
	v, ok := p.kv[k]
	return v, ok
}

func (p mxProbe) Set(_ context.Context, k, v string) error {
	p.note("Set:" + k + "=" + v)
	return nil
}
func (p mxProbe) Acquire(context.Context) (int, error)   { p.note("Acquire"); return 1, nil }
func (p mxProbe) Release(_ context.Context, r int) error { p.note("Release"); return nil }
func (p mxProbe) All(context.Context) ([]string, error)  { p.note("All"); return nil, nil }

// TestMethodExpressionConstructorsDelegate pins the reorder: each *Of
// constructor invokes exactly the method its expression names, with the
// drawn arguments in the right slots, on both sides of the comparison.
func TestMethodExpressionConstructorsDelegate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		act  model.Action[mxIface]
		want string
	}{
		{"WriterOf", action.WriterOf("Put", rapid.Just("v"), mxIface.Put), "Put:v"},
		{"ReaderOf", action.ReaderOf("Get", rapid.Just("k"), mxIface.Get), "Get:k"},
		{"AggregatorOf", action.AggregatorOf("Len", mxIface.Len), "Len"},
		{"DeleterOf", action.DeleterOf("Drop", rapid.Just("k"), mxIface.Drop), "Drop:k"},
		{"LifecycleOf", action.LifecycleOf("Close", mxIface.Close), "Close"},
		{"EvictingReaderOf", action.EvictingReaderOf("Peek", rapid.Just("k"), mxIface.Peek), "Peek:k"},
		{"CompositeWriterOf", action.CompositeWriterOf(
			"Set", rapid.Just("k"), rapid.Just("v"), mxIface.Set,
		), "Set:k=v"},
		{"PoolOf", action.PoolOf("Cycle", mxIface.Acquire, mxIface.Release), "Release"},
		{"StreamOf", action.StreamOf("All", mxIface.All), "All"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var sutCalls, refCalls []string
			sut := mxProbe{calls: &sutCalls, kv: map[string]string{"k": "x"}}
			ref := mxProbe{calls: &refCalls, kv: map[string]string{"k": "x"}}
			rapid.Check(t, func(rt *rapid.T) {
				if res := tc.act.Run(rt, mxIface(sut), mxIface(ref)); res.Err != nil {
					rt.Fatalf("agreeing sides must pass: %v", res.Err)
				}
			})
			for side, calls := range map[string][]string{"sut": sutCalls, "ref": refCalls} {
				if !slices.Contains(calls, tc.want) {
					t.Errorf("%s never saw %q; the expression did not reach its method: %v",
						side, tc.want, calls)
				}
			}
		})
	}
}

// TestMethodExpressionConstructorsCompare pins the other half: the
// comparison is the closure-shaped sibling's, so diverging answers
// still red — the reorder is a spelling, not a bypass.
func TestMethodExpressionConstructorsCompare(t *testing.T) {
	t.Parallel()

	var sutCalls, refCalls []string
	sut := mxProbe{calls: &sutCalls, kv: map[string]string{"k": "a"}}
	ref := mxProbe{calls: &refCalls, kv: map[string]string{"k": "b"}}

	act := action.ReaderOf("Get", rapid.Just("k"), mxIface.Get)
	rapid.Check(t, func(rt *rapid.T) {
		if res := act.Run(rt, mxIface(sut), mxIface(ref)); res.Err == nil {
			rt.Fatal("diverging answers must red the action")
		}
	})
}
