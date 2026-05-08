// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cachetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/gen/bench/testdata/generics"
	"go.thesmos.sh/testkit/gen/bench/testdata/generics/cachetest"
)

type item struct {
	ID   string
	Name string
}

func BenchmarkCacheContract(b *testing.B) {
	factory := func() generics.Cache[string, item] {
		return generics.NewInMemoryCache[string, item](func(v item) string { return v.ID })
	}

	cachetest.BenchmarkCacheContract[string, item](b, factory,
		cachetest.CacheBenchOnGet[string, item](
			bench.ReaderHotPath[generics.Cache[string, item], string, item]("known"),
		),
	)
}
