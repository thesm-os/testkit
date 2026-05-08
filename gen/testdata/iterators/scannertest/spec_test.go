// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package scannertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/testdata/iterators"
	"go.thesmos.sh/testkit/gen/testdata/iterators/scannertest"
	"go.thesmos.sh/testkit/suite"
)

func inmemoryFactory() iterators.Scanner {
	return iterators.NewInMemoryScanner()
}

func stubFactory() iterators.Scanner {
	return scannertest.NewScannerStub(nil,
		scannertest.ScannerStubDelegateTo(inmemoryFactory()))
}

func stubBenchFactory() iterators.Scanner {
	return scannertest.NewScannerStub(nil,
		scannertest.ScannerStubBenchMode(),
		scannertest.ScannerStubDelegateTo(inmemoryFactory()))
}

// --- InMemory ---

func TestScannerContract_InMemory(t *testing.T) {
	t.Parallel()
	scannertest.AssertScannerContract(t, inmemoryFactory,
		scannertest.ScannerOnCount(
			suite.AssertAggregatorConsistent[iterators.Scanner, int](3),
		),
		scannertest.ScannerOnScan(
			suite.AssertStreamCompletes[iterators.Scanner, iterators.Item](),
			suite.AssertStreamRespectsBreak[iterators.Scanner, iterators.Item](),
		),
	)
}

func BenchmarkScannerContract_InMemory(b *testing.B) {
	scannertest.BenchmarkScannerContract(b, inmemoryFactory)
}

// --- Stub+DelegateTo ---

func TestScannerContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	scannertest.AssertScannerContract(t, stubFactory)
}

func BenchmarkScannerContract_StubDelegateTo(b *testing.B) {
	scannertest.BenchmarkScannerContract(b, stubFactory)
}

// --- Stub+BenchMode ---

func BenchmarkScannerContract_StubBenchMode(b *testing.B) {
	scannertest.BenchmarkScannerContract(b, stubBenchFactory)
}
