// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/bench/testdata/basic/storetest"
	"go.thesmos.sh/testkit/gen/suite/testdata/basic"
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
			testkit.BenchReaderHotPath[basic.Store, string, basic.Item]("known-1"),
			testkit.BenchReaderAllocsWithin[basic.Store, string, basic.Item]("known-1", 0),
			testkit.BenchReaderConcurrentThroughput[basic.Store, string, basic.Item]("known-1", 4),
		),

		// Writer: untyped plug-in on Put.
		storetest.StoreBenchOnPut(func(b *testing.B, s basic.Store) {
			b.Run("put-new-item", func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()
				for b.Loop() {
					_ = s.Put(b.Context(), basic.Item{ID: "bench", Name: "bench"})
				}
			})
		}),

		// Lifecycle: untyped plug-in on Ping.
		storetest.StoreBenchOnPing(func(b *testing.B, s basic.Store) {
			b.Run("ping-hot-path", func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()
				for b.Loop() {
					_ = s.Ping(b.Context())
				}
			})
		}),

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
