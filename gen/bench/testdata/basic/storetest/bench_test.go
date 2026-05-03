// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/gen/bench/testdata/basic"
	"go.thesmos.sh/testkit/gen/bench/testdata/basic/storetest"
)

func BenchmarkInMemoryStoreContract(b *testing.B) {
	factory := func() basic.Store { return basic.NewInMemoryStore() }

	storetest.BenchmarkStoreContract(b, factory,
		storetest.StoreBenchPrePopulate(func(ctx context.Context, s basic.Store) {
			_ = s.Put(ctx, basic.Item{ID: "known-1", Name: "test"})
			_ = s.Put(ctx, basic.Item{ID: "known-2", Name: "test-2"})
		}),

		// Reader: typed plug-ins on Get.
		storetest.StoreBenchOnGet(
			bench.ReaderHotPath[basic.Store, string, basic.Item]("known-1"),
			bench.ReaderAllocsWithin[basic.Store, string, basic.Item]("known-1", 0),
			bench.ReaderConcurrentThroughput[basic.Store, string, basic.Item]("known-1", 4),
		),

		// Writer: typed plug-in on Put.
		storetest.StoreBenchOnPut(
			bench.WriterHotPath[basic.Store, basic.Item](
				basic.Item{ID: "bench", Name: "bench"},
			),
		),

		// Lifecycle: typed plug-in on Ping.
		storetest.StoreBenchOnPing(
			bench.LifecycleAllocsWithin[basic.Store](0),
		),

		// Unknown: untyped plug-in on Count.
		storetest.StoreBenchOnCount(func(b *testing.B, s basic.Store) {
			b.Run("count-after-populate", func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()
				for b.Loop() {
					_ = s.Count(b.Context())
				}
			})
		}),

		// Custom benchmark.
		storetest.StoreBenchCustom("put-then-get-round-trip", func(b *testing.B, s basic.Store) {
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				_ = s.Put(b.Context(), basic.Item{ID: "rt", Name: "round-trip"})
				_, _ = s.Get(b.Context(), "rt")
			}
		}),
	)
}
