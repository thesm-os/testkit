// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type mapWriter struct {
	mu   sync.Mutex
	data map[string]string
}

func newMapWriter() *mapWriter {
	return &mapWriter{data: make(map[string]string)}
}

type entry struct {
	Key   string
	Value string
}

func (w *mapWriter) Put(_ context.Context, e entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.data[e.Key] = e.Value
	return nil
}

func (w *mapWriter) Get(_ context.Context, key string) (entry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	v, ok := w.data[key]
	if !ok {
		return entry{}, errNotFound
	}
	return entry{Key: key, Value: v}, nil
}

func writerCtx(t *testing.T) suite.WriterContext[*mapWriter, entry] {
	t.Helper()
	return suite.WriterContext[*mapWriter, entry]{
		T: t,
		WriterBindings: bindings.WriterBindings[*mapWriter, entry]{
			Factory: newMapWriter,
			Call: func(ctx context.Context, w *mapWriter, e entry) error {
				return w.Put(ctx, e)
			},
		},
	}
}

func TestWriter(t *testing.T) {
	t.Parallel()

	t.Run("WriteSucceeds for a valid sample", func(t *testing.T) {
		t.Parallel()
		suite.AssertWriteSucceeds[*mapWriter, entry](
			entry{Key: "a", Value: "alpha"})(writerCtx(t))
	})

	t.Run("WriteIsObservable surfaces the value via the paired reader", func(t *testing.T) {
		t.Parallel()
		suite.AssertWriteIsObservable[*mapWriter, entry, string](
			entry{Key: "a", Value: "alpha"},
			func(e entry) string { return e.Key },
			func(ctx context.Context, w *mapWriter, k string) (entry, error) {
				return w.Get(ctx, k)
			},
		)(writerCtx(t))
	})

	t.Run("WriteRejectInvalid surfaces the sentinel for an invalid sample", func(t *testing.T) {
		t.Parallel()
		// mapWriter accepts everything; use a Call adapter that rejects
		// empty keys.
		ctx := suite.WriterContext[*mapWriter, entry]{
			T: t,
			WriterBindings: bindings.WriterBindings[*mapWriter, entry]{
				Factory: newMapWriter,
				Call: func(_ context.Context, w *mapWriter, e entry) error {
					if e.Key == "" {
						return errNotFound // reuse as "invalid"
					}
					w.data[e.Key] = e.Value
					return nil
				},
			},
		}
		suite.AssertWriteRejectInvalid[*mapWriter, entry](entry{}, errNotFound)(ctx)
	})

	t.Run("WriteOverwrite surfaces the second value via the paired reader", func(t *testing.T) {
		t.Parallel()
		suite.AssertWriteOverwrite[*mapWriter, entry, string](
			entry{Key: "a", Value: "first"},
			entry{Key: "a", Value: "second"},
			func(e entry) string { return e.Key },
			func(ctx context.Context, w *mapWriter, k string) (entry, error) {
				return w.Get(ctx, k)
			},
		)(writerCtx(t))
	})

	t.Run("RespectsContext surfaces ctx.Canceled on cancelled call", func(t *testing.T) {
		t.Parallel()
		ctx := suite.WriterContext[*mapWriter, entry]{
			T: t,
			WriterBindings: bindings.WriterBindings[*mapWriter, entry]{
				Factory: newMapWriter,
				Call: func(c context.Context, _ *mapWriter, _ entry) error {
					return c.Err()
				},
			},
		}
		suite.AssertWriterRespectsContext[*mapWriter, entry](
			entry{Key: "a", Value: "alpha"})(ctx)
	})

	t.Run("Idempotent succeeds on repeat write of the same value", func(t *testing.T) {
		t.Parallel()
		suite.AssertWriterIdempotent[*mapWriter, entry](
			entry{Key: "a", Value: "alpha"})(writerCtx(t))
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertWriterConcurrentSafe[*mapWriter, entry](
			entry{Key: "a", Value: "alpha"}, 4, 50)(writerCtx(t))
	})
}

func TestAssertWriterBaseline(t *testing.T) {
	t.Parallel()
	ctx := suite.WriterContext[*mapWriter, entry]{
		T: t,
		WriterBindings: bindings.WriterBindings[*mapWriter, entry]{
			Factory: newMapWriter,
			Call: func(c context.Context, w *mapWriter, e entry) error {
				if err := c.Err(); err != nil {
					return err
				}
				return w.Put(c, e)
			},
		},
	}
	suite.AssertWriterBaseline[*mapWriter, entry](
		entry{Key: "a", Value: "alpha"})(ctx)
}
