// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"iter"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/suite"
)

// dummy is a minimal test fixture for all context-for constructors.
type dummy struct{ called bool }

func dummyFactory() *dummy { return &dummy{} }

func TestReaderContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.ReaderContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string) (int, error) {
			d.called = true
			return 42, nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	testkit.True(t, ctx.Factory != nil, "Factory wired")
	d := ctx.Factory()
	v, err := ctx.Call(t.Context(), d, "k")
	testkit.NoError(t, err, "Call")
	testkit.Equal(t, v, 42, "Call result")
	testkit.True(t, d.called, "Call routed to closure")
}

func TestReaderNoErrorContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.ReaderNoErrorContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string) int {
			d.called = true
			return 7
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	v := ctx.Call(t.Context(), d, "k")
	testkit.Equal(t, v, 7, "Call result")
	testkit.True(t, d.called, "Call routed")
}

func TestReaderWithBoolContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.ReaderWithBoolContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string) (int, bool) {
			d.called = true
			return 1, true
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	v, ok := ctx.Call(t.Context(), d, "k")
	testkit.Equal(t, v, 1, "value")
	testkit.True(t, ok, "ok")
	testkit.True(t, d.called, "Call routed")
}

func TestLookupContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.LookupContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string) (int, string, bool) {
			d.called = true
			return 1, "meta", true
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	v1, v2, ok := ctx.Call(t.Context(), d, "k")
	testkit.Equal(t, v1, 1, "v1")
	testkit.Equal(t, v2, "meta", "v2")
	testkit.True(t, ok, "ok")
	testkit.True(t, d.called, "Call routed")
}

func TestPointerReaderContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.PointerReaderContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string) *int {
			d.called = true
			v := 99
			return &v
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	got := ctx.Call(t.Context(), d, "k")
	testkit.True(t, got != nil && *got == 99, "pointer result")
	testkit.True(t, d.called, "Call routed")
}

func TestMultiReaderContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.MultiReaderContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string) (int, string, error) {
			d.called = true
			return 1, "two", nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	v1, v2, err := ctx.Call(t.Context(), d, "k")
	testkit.NoError(t, err, "Call")
	testkit.Equal(t, v1, 1, "v1")
	testkit.Equal(t, v2, "two", "v2")
	testkit.True(t, d.called, "Call routed")
}

func TestBatchReaderContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.BatchReaderContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, keys []string) ([]int, error) {
			d.called = true
			out := make([]int, len(keys))
			for i := range keys {
				out[i] = i
			}
			return out, nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	vs, err := ctx.Call(t.Context(), d, []string{"a", "b"})
	testkit.NoError(t, err, "Call")
	testkit.Len(t, vs, 2, "batch result length")
	testkit.True(t, d.called, "Call routed")
}

func TestWriterContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.WriterContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string) error {
			d.called = true
			return nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	testkit.NoError(t, ctx.Call(t.Context(), d, "v"), "Call")
	testkit.True(t, d.called, "Call routed")
}

func TestCompositeWriterContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.CompositeWriterContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string, _ int) error {
			d.called = true
			return nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	testkit.NoError(t, ctx.Call(t.Context(), d, "k", 1), "Call")
	testkit.True(t, d.called, "Call routed")
}

func TestMutatorContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.MutatorContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string) {
			d.called = true
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	ctx.Call(t.Context(), d, "v")
	testkit.True(t, d.called, "Call routed")
}

func TestDeleterContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.DeleterContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string) error {
			d.called = true
			return nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	testkit.NoError(t, ctx.Call(t.Context(), d, "k"), "Call")
	testkit.True(t, d.called, "Call routed")
}

func TestMultiArgWriterContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.MultiArgWriterContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string, _ int, _ bool) error {
			d.called = true
			return nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	testkit.NoError(t, ctx.Call(t.Context(), d, "k", 1, true), "Call")
	testkit.True(t, d.called, "Call routed")
}

func TestAggregatorContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.AggregatorContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy) (int, error) {
			d.called = true
			return 5, nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	v, err := ctx.Call(t.Context(), d)
	testkit.NoError(t, err, "Call")
	testkit.Equal(t, v, 5, "result")
	testkit.True(t, d.called, "Call routed")
}

func TestMultiAggregatorContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.MultiAggregatorContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy) (int, string, error) {
			d.called = true
			return 3, "ok", nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	v1, v2, err := ctx.Call(t.Context(), d)
	testkit.NoError(t, err, "Call")
	testkit.Equal(t, v1, 3, "v1")
	testkit.Equal(t, v2, "ok", "v2")
	testkit.True(t, d.called, "Call routed")
}

func TestStreamContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.StreamContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy) iter.Seq2[int, error] {
			d.called = true
			return func(yield func(int, error) bool) {
				yield(1, nil)
			}
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	seq := ctx.Call(t.Context(), d)
	for v, err := range seq {
		testkit.NoError(t, err, "yield error")
		testkit.Equal(t, v, 1, "yielded value")
	}
	testkit.True(t, d.called, "Call routed")
}

func TestStreamConsumerContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.StreamConsumerContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, _ string) (int, error) {
			d.called = true
			return 10, nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	v, err := ctx.Call(t.Context(), d, "source")
	testkit.NoError(t, err, "Call")
	testkit.Equal(t, v, 10, "result")
	testkit.True(t, d.called, "Call routed")
}

func TestLifecycleContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.LifecycleContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy) error {
			d.called = true
			return nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	testkit.NoError(t, ctx.Call(t.Context(), d), "Call")
	testkit.True(t, d.called, "Call routed")
}

func TestVoidLifecycleContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.VoidLifecycleContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy) {
			d.called = true
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	ctx.Call(t.Context(), d)
	testkit.True(t, d.called, "Call routed")
}

func TestPureContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.PureContextFor(
		t, dummyFactory,
		func(d *dummy) string {
			d.called = true
			return "pure"
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	testkit.Equal(t, ctx.Call(d), "pure", "result")
	testkit.True(t, d.called, "Call routed")
}

func TestPredicateContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.PredicateContextFor(
		t, dummyFactory,
		func(d *dummy) bool {
			d.called = true
			return true
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	testkit.True(t, ctx.Call(d), "result")
	testkit.True(t, d.called, "Call routed")
}

func TestMultiArgWriterVariadicContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.MultiArgWriterVariadicContextFor(
		t, dummyFactory,
		func(_ context.Context, d *dummy, args ...any) error {
			d.called = true
			_ = args
			return nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	err := ctx.Call(t.Context(), d, "a", 1, true)
	testkit.NoError(t, err, "Call")
	testkit.True(t, d.called, "Call routed")
}

func TestPoisonAccessorContextFor(t *testing.T) {
	t.Parallel()
	ctx := suite.PoisonAccessorContextFor(
		t, dummyFactory,
		func(d *dummy) error {
			d.called = true
			return nil
		},
	)
	testkit.True(t, ctx.T == t, "T wired")
	d := ctx.Factory()
	testkit.NoError(t, ctx.Call(d), "Call")
	testkit.True(t, d.called, "Call routed")
}
