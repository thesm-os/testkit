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

func TestAssertReaderWithBoolReturns(t *testing.T) {
	t.Parallel()
	ctx := readerWithBoolCtx(t, map[string]int64{"a": 10})
	suite.AssertReaderWithBoolReturns[*boolMap, string, int64]("a", 10)(ctx)
}

func TestAssertReaderWithBoolMissing(t *testing.T) {
	t.Parallel()
	ctx := readerWithBoolCtx(t, map[string]int64{"a": 10})
	suite.AssertReaderWithBoolMissing[*boolMap, string, int64]("nonexistent")(ctx)
}

func TestAssertReaderWithBoolConsistent(t *testing.T) {
	t.Parallel()
	ctx := readerWithBoolCtx(t, map[string]int64{"a": 10})
	suite.AssertReaderWithBoolConsistent[*boolMap, string, int64]("a", 5)(ctx)
}
