// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
)

var errInjected = errors.New("injected fault")

func TestWithReaderFaults(t *testing.T) {
	t.Parallel()

	base := testkit.ReaderBindings[*mapReader, string, string]{
		Factory: func() *mapReader { return newMapReader(map[string]string{"k": "v"}) },
		Call: func(ctx context.Context, r *mapReader, k string) (string, error) {
			return r.Get(ctx, k)
		},
	}

	t.Run("no faults passes through", func(t *testing.T) {
		t.Parallel()
		b := testkit.WithReaderFaults(base, nil)
		got, err := b.Call(t.Context(), b.Factory(), "k")
		testkit.NoError(t, err, "must pass through")
		testkit.Equal(t, got, "v", "must return value")
	})

	t.Run("counted fault fires", func(t *testing.T) {
		t.Parallel()
		fault := testkit.NewCountedFault[testkit.ReaderCall[*mapReader, string, string]](errInjected, 1)
		b := testkit.WithReaderFaults(base, nil, fault)
		_, err := b.Call(t.Context(), b.Factory(), "k")
		testkit.ErrorIs(t, err, errInjected, "must return injected error")
	})

	t.Run("predicate fault matches key", func(t *testing.T) {
		t.Parallel()
		fault := testkit.NewPredicateFault[testkit.ReaderCall[*mapReader, string, string]](
			errInjected,
			func(c testkit.ReaderCall[*mapReader, string, string]) bool { return c.Key == "fail-me" },
		)
		b := testkit.WithReaderFaults(base, nil, fault)
		_, err := b.Call(t.Context(), b.Factory(), "k")
		testkit.NoError(t, err, "non-matching key must pass through")
		_, err = b.Call(t.Context(), b.Factory(), "fail-me")
		testkit.ErrorIs(t, err, errInjected, "matching key must fire")
	})

	t.Run("factory preserved", func(t *testing.T) {
		t.Parallel()
		fault := testkit.NewCountedFault[testkit.ReaderCall[*mapReader, string, string]](errInjected, 1)
		b := testkit.WithReaderFaults(base, nil, fault)
		testkit.True(t, b.Factory() != nil, "factory must produce impl")
	})
}

func TestWithWriterFaults(t *testing.T) {
	t.Parallel()

	base := testkit.WriterBindings[*mapWriter, entry]{
		Factory: newMapWriter,
		Call: func(ctx context.Context, w *mapWriter, e entry) error {
			return w.Put(ctx, e)
		},
	}

	t.Run("no faults passes through", func(t *testing.T) {
		t.Parallel()
		b := testkit.WithWriterFaults(base, nil)
		err := b.Call(t.Context(), b.Factory(), entry{Key: "a", Value: "v"})
		testkit.NoError(t, err, "must pass through")
	})

	t.Run("counted fault fires", func(t *testing.T) {
		t.Parallel()
		fault := testkit.NewCountedFault[testkit.WriterCall[*mapWriter, entry]](errInjected, 1)
		b := testkit.WithWriterFaults(base, nil, fault)
		err := b.Call(t.Context(), b.Factory(), entry{Key: "a", Value: "v"})
		testkit.ErrorIs(t, err, errInjected, "must return injected error")
	})

	t.Run("predicate fault matches value", func(t *testing.T) {
		t.Parallel()
		fault := testkit.NewPredicateFault[testkit.WriterCall[*mapWriter, entry]](
			errInjected,
			func(c testkit.WriterCall[*mapWriter, entry]) bool { return c.Val.Key == "bad" },
		)
		b := testkit.WithWriterFaults(base, nil, fault)
		err := b.Call(t.Context(), b.Factory(), entry{Key: "good", Value: "v"})
		testkit.NoError(t, err, "non-matching value must pass through")
		err = b.Call(t.Context(), b.Factory(), entry{Key: "bad", Value: "v"})
		testkit.ErrorIs(t, err, errInjected, "matching value must fire")
	})
}

func TestWithDeleterFaults(t *testing.T) {
	t.Parallel()

	base := testkit.DeleterBindings[*delStore, string]{
		Factory: newDelStore,
		Call: func(ctx context.Context, s *delStore, k string) error {
			return s.Delete(ctx, k)
		},
	}

	t.Run("no faults passes through", func(t *testing.T) {
		t.Parallel()
		b := testkit.WithDeleterFaults(base, nil)
		err := b.Call(t.Context(), b.Factory(), "existing")
		testkit.NoError(t, err, "must pass through")
	})

	t.Run("counted fault fires", func(t *testing.T) {
		t.Parallel()
		fault := testkit.NewCountedFault[testkit.DeleterCall[*delStore, string]](errInjected, 1)
		b := testkit.WithDeleterFaults(base, nil, fault)
		err := b.Call(t.Context(), b.Factory(), "existing")
		testkit.ErrorIs(t, err, errInjected, "must return injected error")
	})

	t.Run("predicate fault matches key", func(t *testing.T) {
		t.Parallel()
		fault := testkit.NewPredicateFault[testkit.DeleterCall[*delStore, string]](
			errInjected,
			func(c testkit.DeleterCall[*delStore, string]) bool { return c.Key == "protected" },
		)
		b := testkit.WithDeleterFaults(base, nil, fault)
		err := b.Call(t.Context(), b.Factory(), "existing")
		testkit.NoError(t, err, "non-matching key must pass through")
		err = b.Call(t.Context(), b.Factory(), "protected")
		testkit.ErrorIs(t, err, errInjected, "matching key must fire")
	})
}

func TestWithAggregatorFaults(t *testing.T) {
	t.Parallel()

	base := testkit.AggregatorBindings[*itemCounter, int]{
		Factory: func() *itemCounter { return newItemCounter(42) },
		Call: func(ctx context.Context, c *itemCounter) (int, error) {
			return c.Count(ctx)
		},
	}

	t.Run("no faults passes through", func(t *testing.T) {
		t.Parallel()
		b := testkit.WithAggregatorFaults(base, nil)
		got, err := b.Call(t.Context(), b.Factory())
		testkit.NoError(t, err, "must pass through")
		testkit.Equal(t, got, 42, "must return value")
	})

	t.Run("counted fault fires", func(t *testing.T) {
		t.Parallel()
		fault := testkit.NewCountedFault[testkit.AggregatorCall[*itemCounter, int]](errInjected, 1)
		b := testkit.WithAggregatorFaults(base, nil, fault)
		got, err := b.Call(t.Context(), b.Factory())
		testkit.ErrorIs(t, err, errInjected, "must return injected error")
		testkit.Equal(t, got, 0, "must return zero on fault")
	})

	t.Run("predicate fault matches context", func(t *testing.T) {
		t.Parallel()
		type ctxKey struct{}
		fault := testkit.NewPredicateFault[testkit.AggregatorCall[*itemCounter, int]](
			errInjected,
			func(c testkit.AggregatorCall[*itemCounter, int]) bool {
				return c.Ctx.Value(ctxKey{}) == "trigger"
			},
		)
		b := testkit.WithAggregatorFaults(base, nil, fault)
		_, err := b.Call(t.Context(), b.Factory())
		testkit.NoError(t, err, "plain context must pass through")
		triggerCtx := context.WithValue(t.Context(), ctxKey{}, "trigger")
		_, err = b.Call(triggerCtx, b.Factory())
		testkit.ErrorIs(t, err, errInjected, "trigger context must fire")
	})
}

func TestWithLifecycleFaults(t *testing.T) {
	t.Parallel()

	base := testkit.LifecycleBindings[*lifecycle]{
		Factory: newLifecycle,
		Call: func(ctx context.Context, l *lifecycle) error {
			return l.Open(ctx)
		},
	}

	t.Run("no faults passes through", func(t *testing.T) {
		t.Parallel()
		b := testkit.WithLifecycleFaults(base, nil)
		err := b.Call(t.Context(), b.Factory())
		testkit.NoError(t, err, "must pass through")
	})

	t.Run("counted fault fires", func(t *testing.T) {
		t.Parallel()
		fault := testkit.NewCountedFault[testkit.LifecycleCall[*lifecycle]](errInjected, 1)
		b := testkit.WithLifecycleFaults(base, nil, fault)
		err := b.Call(t.Context(), b.Factory())
		testkit.ErrorIs(t, err, errInjected, "must return injected error")
	})

	t.Run("predicate fault matches context", func(t *testing.T) {
		t.Parallel()
		type ctxKey struct{}
		fault := testkit.NewPredicateFault[testkit.LifecycleCall[*lifecycle]](
			errInjected,
			func(c testkit.LifecycleCall[*lifecycle]) bool {
				return c.Ctx.Value(ctxKey{}) == "trigger"
			},
		)
		b := testkit.WithLifecycleFaults(base, nil, fault)
		err := b.Call(t.Context(), b.Factory())
		testkit.NoError(t, err, "plain context must pass through")
		triggerCtx := context.WithValue(t.Context(), ctxKey{}, "trigger")
		err = b.Call(triggerCtx, b.Factory())
		testkit.ErrorIs(t, err, errInjected, "trigger context must fire")
	})
}
