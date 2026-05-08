// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package processortest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/testdata/interfaces"
	"go.thesmos.sh/testkit/gen/testdata/interfaces/processortest"
)

func inmemoryFactory() interfaces.Processor { return interfaces.NewInMemoryProcessor() }
func stubFactory() interfaces.Processor {
	return processortest.NewProcessorStub(nil, processortest.ProcessorStubDelegateTo(inmemoryFactory()))
}
func stubBenchFactory() interfaces.Processor {
	return processortest.NewProcessorStub(nil, processortest.ProcessorStubBenchMode(), processortest.ProcessorStubDelegateTo(inmemoryFactory()))
}

func TestProcessorContract_InMemory(t *testing.T) {
	t.Parallel()
	processortest.AssertProcessorContract(t, inmemoryFactory)
}
func BenchmarkProcessorContract_InMemory(b *testing.B) {
	processortest.BenchmarkProcessorContract(b, inmemoryFactory)
}
func TestProcessorContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	processortest.AssertProcessorContract(t, stubFactory)
}
func BenchmarkProcessorContract_StubDelegateTo(b *testing.B) {
	processortest.BenchmarkProcessorContract(b, stubFactory)
}
func BenchmarkProcessorContract_StubBenchMode(b *testing.B) {
	processortest.BenchmarkProcessorContract(b, stubBenchFactory)
}
