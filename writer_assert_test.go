// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
)

type mapWriter struct {
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
	w.data[e.Key] = e.Value
	return nil
}

func (w *mapWriter) Get(_ context.Context, key string) (entry, error) {
	v, ok := w.data[key]
	if !ok {
		return entry{}, errNotFound
	}
	return entry{Key: key, Value: v}, nil
}

func writerCtx(t *testing.T) testkit.WriterContext[*mapWriter, entry] {
	t.Helper()
	return testkit.WriterContext[*mapWriter, entry]{
		T:       t,
		Factory: newMapWriter,
		Call: func(w *mapWriter, ctx context.Context, e entry) error {
			return w.Put(ctx, e)
		},
	}
}

func TestAssertWriteSucceeds(t *testing.T) {
	t.Parallel()
	ctx := writerCtx(t)
	testkit.AssertWriteSucceeds[*mapWriter, entry](entry{Key: "a", Value: "alpha"})(ctx)
}

func TestAssertWriteIsObservable(t *testing.T) {
	t.Parallel()
	ctx := writerCtx(t)
	testkit.AssertWriteIsObservable[*mapWriter, entry, string](
		entry{Key: "a", Value: "alpha"},
		func(e entry) string { return e.Key },
		func(w *mapWriter, ctx context.Context, k string) (entry, error) {
			return w.Get(ctx, k)
		},
	)(ctx)
}

func TestAssertWriteRejectInvalid(t *testing.T) {
	t.Parallel()
	// mapWriter accepts everything, so use a custom call that rejects empty keys.
	ctx := testkit.WriterContext[*mapWriter, entry]{
		T:       t,
		Factory: newMapWriter,
		Call: func(w *mapWriter, _ context.Context, e entry) error {
			if e.Key == "" {
				return errNotFound // reuse as "invalid"
			}
			w.data[e.Key] = e.Value
			return nil
		},
	}
	testkit.AssertWriteRejectInvalid[*mapWriter, entry](entry{}, errNotFound)(ctx)
}

func TestAssertWriteOverwrite(t *testing.T) {
	t.Parallel()
	ctx := writerCtx(t)
	testkit.AssertWriteOverwrite[*mapWriter, entry, string](
		entry{Key: "a", Value: "first"},
		entry{Key: "a", Value: "second"},
		func(e entry) string { return e.Key },
		func(w *mapWriter, ctx context.Context, k string) (entry, error) {
			return w.Get(ctx, k)
		},
	)(ctx)
}
