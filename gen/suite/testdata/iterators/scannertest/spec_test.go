// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package scannertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/suite/testdata/iterators"
	"go.thesmos.sh/testkit/gen/suite/testdata/iterators/scannertest"
)

func TestInMemoryScannerContract(t *testing.T) {
	t.Parallel()
	factory := func() iterators.Scanner { return iterators.NewInMemoryScanner() }

	scannertest.AssertScannerContract(t, factory)
}
