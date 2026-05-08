// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package countertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/testdata/voidctx"
	"go.thesmos.sh/testkit/gen/testdata/voidctx/countertest"
)

func inmemoryFactory() voidctx.Counter { return voidctx.NewInMemoryCounter("test") }
func stubFactory() voidctx.Counter {
	return countertest.NewCounterStub(nil, countertest.CounterStubDelegateTo(inmemoryFactory()))
}
func stubBenchFactory() voidctx.Counter {
	return countertest.NewCounterStub(nil, countertest.CounterStubBenchMode(), countertest.CounterStubDelegateTo(inmemoryFactory()))
}

func TestCounterContract_InMemory(t *testing.T) {
	t.Parallel()
	countertest.AssertCounterContract(t, inmemoryFactory)
}
func BenchmarkCounterContract_InMemory(b *testing.B) {
	countertest.BenchmarkCounterContract(b, inmemoryFactory)
}
func TestCounterContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	countertest.AssertCounterContract(t, stubFactory)
}
func BenchmarkCounterContract_StubDelegateTo(b *testing.B) {
	countertest.BenchmarkCounterContract(b, stubFactory)
}
func BenchmarkCounterContract_StubBenchMode(b *testing.B) {
	countertest.BenchmarkCounterContract(b, stubBenchFactory)
}
