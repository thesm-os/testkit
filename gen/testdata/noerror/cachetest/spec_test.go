// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cachetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/testdata/noerror"
	"go.thesmos.sh/testkit/gen/testdata/noerror/cachetest"
)

func inmemoryFactory() noerror.Cache { return noerror.NewInMemoryCache() }
func stubFactory() noerror.Cache {
	return cachetest.NewCacheStub(nil, cachetest.CacheStubDelegateTo(inmemoryFactory()))
}
func stubBenchFactory() noerror.Cache {
	return cachetest.NewCacheStub(nil, cachetest.CacheStubBenchMode(), cachetest.CacheStubDelegateTo(inmemoryFactory()))
}

func TestCacheContract_InMemory(t *testing.T) {
	t.Parallel()
	cachetest.AssertCacheContract(t, inmemoryFactory)
}
func BenchmarkCacheContract_InMemory(b *testing.B) {
	cachetest.BenchmarkCacheContract(b, inmemoryFactory)
}
func TestCacheContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	cachetest.AssertCacheContract(t, stubFactory)
}
func BenchmarkCacheContract_StubDelegateTo(b *testing.B) {
	cachetest.BenchmarkCacheContract(b, stubFactory)
}
func BenchmarkCacheContract_StubBenchMode(b *testing.B) {
	cachetest.BenchmarkCacheContract(b, stubBenchFactory)
}
