// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cachetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/gen/testdata/generics"
	"go.thesmos.sh/testkit/gen/testdata/generics/cachetest"
	"go.thesmos.sh/testkit/suite"
)

type item struct {
	ID   string
	Name string
}

func inmemoryFactory() generics.Cache[string, item] {
	c := generics.NewInMemoryCache[string, item](func(v item) string { return v.ID })
	_ = c.Put(context.Background(), item{ID: "seed", Name: "seed"})
	return c
}

func stubFactory() generics.Cache[string, item] {
	return cachetest.NewCacheStub[string, item](nil,
		cachetest.CacheStubDelegateTo[string, item](inmemoryFactory()))
}

func stubBenchFactory() generics.Cache[string, item] {
	return cachetest.NewCacheStub[string, item](nil,
		cachetest.CacheStubBenchMode[string, item](),
		cachetest.CacheStubDelegateTo[string, item](inmemoryFactory()))
}

func suiteOpts() []cachetest.CacheOption[string, item] {
	return []cachetest.CacheOption[string, item]{
		cachetest.CacheOnGet[string, item](
			suite.AssertReturnsSentinel[generics.Cache[string, item], string, item](
				"nonexistent", generics.ErrNotFound),
		),
		cachetest.CacheOnDelete[string, item](
			suite.AssertDeleteSucceeds[generics.Cache[string, item], string]("seed"),
		),
		cachetest.CacheOnLoad[string, item](
			suite.AssertReaderWithBoolMissing[generics.Cache[string, item], string, item](
				"nonexistent"),
		),
	}
}

func benchOpts() []cachetest.CacheBenchOption[string, item] {
	return []cachetest.CacheBenchOption[string, item]{
		cachetest.CacheBenchOnGet[string, item](
			bench.ReaderHotPath[generics.Cache[string, item], string, item]("seed"),
		),
	}
}

// --- InMemory ---

func TestCacheContract_InMemory(t *testing.T) {
	t.Parallel()
	cachetest.AssertCacheContract[string, item](t, inmemoryFactory, suiteOpts()...)
}

func BenchmarkCacheContract_InMemory(b *testing.B) {
	cachetest.BenchmarkCacheContract[string, item](b, inmemoryFactory, benchOpts()...)
}

// --- Stub+DelegateTo ---

func TestCacheContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	cachetest.AssertCacheContract[string, item](t, stubFactory, suiteOpts()...)
}

func BenchmarkCacheContract_StubDelegateTo(b *testing.B) {
	cachetest.BenchmarkCacheContract[string, item](b, stubFactory, benchOpts()...)
}

// --- Stub+BenchMode ---

func BenchmarkCacheContract_StubBenchMode(b *testing.B) {
	cachetest.BenchmarkCacheContract[string, item](b, stubBenchFactory, benchOpts()...)
}
