// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package scannertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/suite/testdata/iterators"
	"go.thesmos.sh/testkit/gen/suite/testdata/iterators/scannertest"
)

func TestInMemoryScannerContract(t *testing.T) {
	t.Parallel()
	factory := func() iterators.Scanner { return iterators.NewInMemoryScanner() }

	scannertest.AssertScannerContract(t, factory,
		scannertest.ScannerOnKeys(
			testkit.AssertStreamCompletes[iterators.Scanner, string](),
		),
		scannertest.ScannerOnScan(
			testkit.AssertStreamCompletes[iterators.Scanner, iterators.Item](),
			testkit.AssertStreamRespectsBreak[iterators.Scanner, iterators.Item](),
		),
		scannertest.ScannerOnCount(
			testkit.AssertAggregatorConsistent[iterators.Scanner, int](3),
		),
	)
}
