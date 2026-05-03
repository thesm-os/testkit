// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cachetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/suite/testdata/nocontext"
	"go.thesmos.sh/testkit/gen/suite/testdata/nocontext/cachetest"
	"go.thesmos.sh/testkit/suite"
)

func TestInMemoryCacheContract(t *testing.T) {
	t.Parallel()
	factory := func() nocontext.Cache { return nocontext.NewInMemoryCache() }

	cachetest.AssertCacheContract(t, factory,
		cachetest.CacheOnGet(func(t *testing.T, c nocontext.Cache) {
			_, _ = c.Get("test")
		}),
		cachetest.CacheOnSet(func(t *testing.T, c nocontext.Cache) {
			_ = c.Set("k", "v")
		}),
		cachetest.CacheOnLen(
			suite.AssertDeterministic[nocontext.Cache, int](3),
		),
	)
}
