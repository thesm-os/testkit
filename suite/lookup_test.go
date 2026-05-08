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

func TestAssertLookupReturns(t *testing.T) {
	t.Parallel()
	ctx := lookupCtx(t)
	suite.AssertLookupReturns[*lookupMap, string, int64, meta]("a", 10)(ctx)
}

func TestAssertLookupMissing(t *testing.T) {
	t.Parallel()
	ctx := lookupCtx(t)
	suite.AssertLookupMissing[*lookupMap, string, int64, meta]("nonexistent")(ctx)
}
