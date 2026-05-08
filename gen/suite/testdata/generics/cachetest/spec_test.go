// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cachetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/gen/suite/testdata/generics"
	"go.thesmos.sh/testkit/gen/suite/testdata/generics/cachetest"
	"go.thesmos.sh/testkit/suite"
)

type item struct {
	ID   string
	Name string
}

func TestCacheContract(t *testing.T) {
	t.Parallel()
	factory := func() generics.Cache[string, item] {
		return generics.NewInMemoryCache[string, item](func(v item) string { return v.ID })
	}

	cachetest.AssertCacheContract[string, item](t, factory,
		cachetest.CachePrePopulate[string, item](func(ctx context.Context, c generics.Cache[string, item]) {
			_ = c.Put(ctx, item{ID: "known", Name: "test"})
		}),
		cachetest.CacheOnGet[string, item](
			suite.AssertReturnsSentinel[generics.Cache[string, item], string, item](
				"nonexistent", generics.ErrNotFound,
			),
		),
		cachetest.CacheOnDelete[string, item](
			suite.AssertDeleteSucceeds[generics.Cache[string, item], string]("known"),
		),
	)
}
