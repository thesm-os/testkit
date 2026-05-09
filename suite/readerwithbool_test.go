// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type boolMap struct{ data map[string]int64 }

func newBoolMap(data map[string]int64) *boolMap { return &boolMap{data: data} }

func (m *boolMap) Load(_ context.Context, key string) (int64, bool) {
	v, ok := m.data[key]
	return v, ok
}

func readerWithBoolCtx(t *testing.T, data map[string]int64) suite.ReaderWithBoolContext[*boolMap, string, int64] {
	t.Helper()
	return suite.ReaderWithBoolContext[*boolMap, string, int64]{
		T: t,
		ReaderWithBoolBindings: bindings.ReaderWithBoolBindings[*boolMap, string, int64]{
			Factory: func() *boolMap { return newBoolMap(data) },
			Call: func(ctx context.Context, m *boolMap, k string) (int64, bool) {
				return m.Load(ctx, k)
			},
		},
	}
}

func TestReaderWithBool(t *testing.T) {
	t.Parallel()
	data := map[string]int64{"a": 10}

	t.Run("Returns surfaces (value, true) for a known key", func(t *testing.T) {
		t.Parallel()
		suite.AssertReaderWithBoolReturns[*boolMap, string, int64](
			"a", 10)(readerWithBoolCtx(t, data))
	})

	t.Run("Missing surfaces (zero, false) for an unknown key", func(t *testing.T) {
		t.Parallel()
		suite.AssertReaderWithBoolMissing[*boolMap, string, int64](
			"nonexistent")(readerWithBoolCtx(t, data))
	})

	t.Run("Consistent yields equal (value, ok) across N calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertReaderWithBoolConsistent[*boolMap, string, int64](
			"a", 5)(readerWithBoolCtx(t, data))
	})

	t.Run("RespectsContext surfaces (zero, false) under cancelled ctx", func(t *testing.T) {
		t.Parallel()
		ctx := suite.ReaderWithBoolContext[*boolMap, string, int64]{
			T: t,
			ReaderWithBoolBindings: bindings.ReaderWithBoolBindings[*boolMap, string, int64]{
				Factory: func() *boolMap { return newBoolMap(data) },
				Call: func(c context.Context, m *boolMap, k string) (int64, bool) {
					if c.Err() != nil {
						return 0, false
					}
					return m.Load(c, k)
				},
			},
		}
		suite.AssertReaderWithBoolRespectsContext[*boolMap, string, int64]("a")(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertReaderWithBoolConcurrentSafe[*boolMap, string, int64](
			"a", 4, 50)(readerWithBoolCtx(t, data))
	})
}
