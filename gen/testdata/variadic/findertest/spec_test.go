// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package findertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/testdata/variadic"
	"go.thesmos.sh/testkit/gen/testdata/variadic/findertest"
)

func inmemoryFactory() variadic.Finder { return variadic.NewInMemoryFinder() }
func stubFactory() variadic.Finder {
	return findertest.NewFinderStub(nil, findertest.FinderStubDelegateTo(inmemoryFactory()))
}
func stubBenchFactory() variadic.Finder {
	return findertest.NewFinderStub(nil, findertest.FinderStubBenchMode(), findertest.FinderStubDelegateTo(inmemoryFactory()))
}

func TestFinderContract_InMemory(t *testing.T) {
	t.Parallel()
	findertest.AssertFinderContract(t, inmemoryFactory)
}
func BenchmarkFinderContract_InMemory(b *testing.B) {
	findertest.BenchmarkFinderContract(b, inmemoryFactory)
}
func TestFinderContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	findertest.AssertFinderContract(t, stubFactory)
}
func BenchmarkFinderContract_StubDelegateTo(b *testing.B) {
	findertest.BenchmarkFinderContract(b, stubFactory)
}
func BenchmarkFinderContract_StubBenchMode(b *testing.B) {
	findertest.BenchmarkFinderContract(b, stubBenchFactory)
}
