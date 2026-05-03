// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
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

func TestAssertWriteSucceeds(t *testing.T) {
	t.Parallel()
	ctx := writerCtx(t)
	suite.AssertWriteSucceeds[*mapWriter, entry](entry{Key: "a", Value: "alpha"})(ctx)
}

func TestAssertWriteIsObservable(t *testing.T) {
	t.Parallel()
	ctx := writerCtx(t)
	suite.AssertWriteIsObservable[*mapWriter, entry, string](
		entry{Key: "a", Value: "alpha"},
		func(e entry) string { return e.Key },
		func(ctx context.Context, w *mapWriter, k string) (entry, error) {
			return w.Get(ctx, k)
		},
	)(ctx)
}

func TestAssertWriteRejectInvalid(t *testing.T) {
	t.Parallel()
	// mapWriter accepts everything, so use a custom call that rejects empty keys.
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
}

func TestAssertWriteOverwrite(t *testing.T) {
	t.Parallel()
	ctx := writerCtx(t)
	suite.AssertWriteOverwrite[*mapWriter, entry, string](
		entry{Key: "a", Value: "first"},
		entry{Key: "a", Value: "second"},
		func(e entry) string { return e.Key },
		func(ctx context.Context, w *mapWriter, k string) (entry, error) {
			return w.Get(ctx, k)
		},
	)(ctx)
}
