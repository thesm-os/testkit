// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/basic"
	"go.thesmos.sh/testkit/generator/testdata/basic/storetest"
)

// BenchmarkPerf closes the loop on the //testkit:allocs and
// //testkit:latency budget gates. The factory uses
// [basic.NewInMemoryPerf] which pre-seeds "test-key" → Item{ID:
// "test-id"} so the //testkit:allocs 0 gate observes the success
// path (not an ErrNotFound construction that would blow the alloc
// budget).
func BenchmarkPerf(b *testing.B) {
	storetest.BenchmarkPerfContract(b, func() basic.Perf {
		return basic.NewInMemoryPerf()
	})
}
