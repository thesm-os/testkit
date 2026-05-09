// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type meta struct{ Version string }

type lookupMap struct {
	values map[string]int64
	meta   map[string]meta
}

func newLookupMap() *lookupMap {
	return &lookupMap{
		values: map[string]int64{"a": 10},
		meta:   map[string]meta{"a": {Version: "v1"}},
	}
}

func (m *lookupMap) Inspect(_ context.Context, key string) (int64, meta, bool) {
	v, ok := m.values[key]
	if !ok {
		return 0, meta{}, false
	}
	return v, m.meta[key], true
}

func lookupCtx(t *testing.T) suite.LookupContext[*lookupMap, string, int64, meta] {
	t.Helper()
	return suite.LookupContext[*lookupMap, string, int64, meta]{
		T: t,
		LookupBindings: bindings.LookupBindings[*lookupMap, string, int64, meta]{
			Factory: newLookupMap,
			Call: func(ctx context.Context, m *lookupMap, k string) (int64, meta, bool) {
				return m.Inspect(ctx, k)
			},
		},
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()

	t.Run("Returns surfaces the value for a known key", func(t *testing.T) {
		t.Parallel()
		suite.AssertLookupReturns[*lookupMap, string, int64, meta]("a", 10)(lookupCtx(t))
	})

	t.Run("Missing surfaces ok=false for an unknown key", func(t *testing.T) {
		t.Parallel()
		suite.AssertLookupMissing[*lookupMap, string, int64, meta]("nonexistent")(lookupCtx(t))
	})

	t.Run("Consistent yields equal values across N calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertLookupConsistent[*lookupMap, string, int64, meta]("a", 4)(lookupCtx(t))
	})

	t.Run("RespectsContext surfaces ok=false on cancelled call", func(t *testing.T) {
		t.Parallel()
		ctx := suite.LookupContext[*lookupMap, string, int64, meta]{
			T: t,
			LookupBindings: bindings.LookupBindings[*lookupMap, string, int64, meta]{
				Factory: newLookupMap,
				Call: func(c context.Context, m *lookupMap, k string) (int64, meta, bool) {
					if c.Err() != nil {
						return 0, meta{}, false
					}
					return m.Inspect(c, k)
				},
			},
		}
		suite.AssertLookupRespectsContext[*lookupMap, string, int64, meta]("a")(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertLookupConcurrentSafe[*lookupMap, string, int64, meta](
			"a", 4, 50)(lookupCtx(t))
	})
}

func TestAssertLookupBaseline(t *testing.T) {
	t.Parallel()
	ctx := suite.LookupContext[*lookupMap, string, int64, meta]{
		T: t,
		LookupBindings: bindings.LookupBindings[*lookupMap, string, int64, meta]{
			Factory: newLookupMap,
			Call: func(c context.Context, m *lookupMap, k string) (int64, meta, bool) {
				if c.Err() != nil {
					return 0, meta{}, false
				}
				return m.Inspect(c, k)
			},
		},
	}
	suite.AssertLookupBaseline[*lookupMap, string, int64, meta](
		"a", 10, "missing")(ctx)
}
