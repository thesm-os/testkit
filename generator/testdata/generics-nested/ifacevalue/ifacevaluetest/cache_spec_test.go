// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ifacevaluetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics-nested/ifacevalue"
)

// TestReaderCacheContract closes the e2e loop on the interface-
// typed value cache. Verifies the suite generator emits valid Go
// for io.Reader value types.
func TestReaderCacheContract(t *testing.T) {
	t.Parallel()
	AssertReaderCacheContract(t, func() ifacevalue.ReaderCache {
		return ifacevalue.NewInMem()
	})
}
