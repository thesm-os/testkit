// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package scannertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/suite/testdata/iterators"
	"go.thesmos.sh/testkit/gen/suite/testdata/iterators/scannertest"
	"go.thesmos.sh/testkit/suite"
)

func TestInMemoryScannerContract(t *testing.T) {
	t.Parallel()
	factory := func() iterators.Scanner { return iterators.NewInMemoryScanner() }

	scannertest.AssertScannerContract(t, factory,
		scannertest.ScannerOnKeys(
			suite.AssertStreamCompletes[iterators.Scanner, string](),
		),
		scannertest.ScannerOnScan(
			suite.AssertStreamCompletes[iterators.Scanner, iterators.Item](),
			suite.AssertStreamRespectsBreak[iterators.Scanner, iterators.Item](),
		),
		scannertest.ScannerOnCount(
			suite.AssertAggregatorConsistent[iterators.Scanner, int](3),
		),
	)
}
