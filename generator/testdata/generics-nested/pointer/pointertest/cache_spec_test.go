// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package pointertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics-nested/pointer"
)

// TestContainerCacheContract closes the e2e loop on the
// pointer-typed value cache. Verifies the suite generator emits
// valid Go for *Container[int] value types.
func TestContainerCacheContract(t *testing.T) {
	t.Parallel()
	AssertContainerCacheContract(t, func() pointer.ContainerCache {
		return pointer.NewInMem()
	})
}
