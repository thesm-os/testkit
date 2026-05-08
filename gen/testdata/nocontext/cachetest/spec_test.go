// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cachetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/testdata/nocontext"
	"go.thesmos.sh/testkit/gen/testdata/nocontext/cachetest"
)

func inmemoryFactory() nocontext.Cache {
	c := nocontext.NewInMemoryCache()
	_ = c.Set("seed", "value")
	return c
}

func stubFactory() nocontext.Cache {
	return cachetest.NewCacheStub(nil,
		cachetest.CacheStubDelegateTo(inmemoryFactory()))
}

func stubBenchFactory() nocontext.Cache {
	return cachetest.NewCacheStub(nil,
		cachetest.CacheStubBenchMode(),
		cachetest.CacheStubDelegateTo(inmemoryFactory()))
}

// --- InMemory ---

func TestCacheContract_InMemory(t *testing.T) {
	t.Parallel()
	cachetest.AssertCacheContract(t, inmemoryFactory)
}

func BenchmarkCacheContract_InMemory(b *testing.B) {
	cachetest.BenchmarkCacheContract(b, inmemoryFactory)
}

// --- Stub+DelegateTo ---

func TestCacheContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	cachetest.AssertCacheContract(t, stubFactory)
}

func BenchmarkCacheContract_StubDelegateTo(b *testing.B) {
	cachetest.BenchmarkCacheContract(b, stubFactory)
}

// --- Stub+BenchMode ---

func BenchmarkCacheContract_StubBenchMode(b *testing.B) {
	cachetest.BenchmarkCacheContract(b, stubBenchFactory)
}
