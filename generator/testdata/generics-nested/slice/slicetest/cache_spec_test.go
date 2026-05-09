// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package slicetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics-nested/slice"
)

// TestByteCacheContract closes the e2e loop on the slice-typed
// value cache. Verifies the suite generator emits valid Go for
// []byte value types.
func TestByteCacheContract(t *testing.T) {
	t.Parallel()
	AssertByteCacheContract(t, func() slice.ByteCache {
		return slice.NewInMem()
	})
}
