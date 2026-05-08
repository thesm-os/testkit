// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package streamtest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/testdata/voidpure"
	"go.thesmos.sh/testkit/gen/testdata/voidpure/streamtest"
)

func inmemoryFactory() voidpure.Stream { return voidpure.NewInMemoryStream() }
func stubFactory() voidpure.Stream {
	return streamtest.NewStreamStub(nil, streamtest.StreamStubDelegateTo(inmemoryFactory()))
}
func stubBenchFactory() voidpure.Stream {
	return streamtest.NewStreamStub(nil, streamtest.StreamStubBenchMode(), streamtest.StreamStubDelegateTo(inmemoryFactory()))
}

func TestStreamContract_InMemory(t *testing.T) {
	t.Parallel()
	streamtest.AssertStreamContract(t, inmemoryFactory)
}
func BenchmarkStreamContract_InMemory(b *testing.B) {
	streamtest.BenchmarkStreamContract(b, inmemoryFactory)
}
func TestStreamContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	streamtest.AssertStreamContract(t, stubFactory)
}
func BenchmarkStreamContract_StubDelegateTo(b *testing.B) {
	streamtest.BenchmarkStreamContract(b, stubFactory)
}
func BenchmarkStreamContract_StubBenchMode(b *testing.B) {
	streamtest.BenchmarkStreamContract(b, stubBenchFactory)
}
