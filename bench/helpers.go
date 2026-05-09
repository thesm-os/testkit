// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"
)

// SubtestKey renders an arbitrary value into a benchmark-safe
// subtest segment. Strings, numbers, and other atomic types render
// via `fmt.Sprintf("%v", v)` directly. Composite values (structs,
// maps, slices) are sanitized: `fmt.Sprintf("%v", item)` produces
// strings like `{test-id }` whose embedded spaces become awkward
// underscores once the testing framework escapes them; SubtestKey
// strips the bracing and replaces internal whitespace with `-`,
// yielding `test-id`.
//
// Exported because consumers writing custom drivers want the same
// rendering when they construct subtest names from sample values.
func SubtestKey(v any) string {
	s := fmt.Sprintf("%v", v)
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	s = strings.TrimSpace(s)
	if s == "" {
		return "_"
	}
	return strings.Join(strings.Fields(s), "-")
}

// errSink is the package-level error sink. Assigning a method's
// returned error here prevents the compiler from eliding the call
// in benchmark contexts the [testing.B.Loop] keep-alive contract
// doesn't cover (notably [testing.B.RunParallel] and
// [testing.AllocsPerRun] closures).
//
// The sink is intentionally write-only — read access is absent so
// the compiler can't prove the write is unobserved. Assigning to
// errSink is itself zero-allocation: error is an interface type, so
// the assignment is a pointer copy. Value returns rely on the
// b.Loop semantics or interface dispatch through ctx.Call to defeat
// elision; package-level value sinks would box and pollute
// AllocsWithin counts.
//
//nolint:gochecknoglobals,unused // benchmark elision sink
var errSink error

// HotPath drives a single-goroutine measurement of `call` under the
// canonical b.Loop discipline: ResetTimer + ReportAllocs + b.Loop.
// Reports ns/op and allocs/op for the configured run.
//
// Intended for both the shape-primitive vocabulary in this package
// and consumer-written benchmark drivers for shapes we don't cover.
// Hoist per-call invariants (impl, sample values, b.Context()) to the
// caller closure so each iteration is the minimum work the impl
// performs.
//
// b.Loop's keep-alive semantics defeat dead-code elimination for any
// values the closure references — assigning to a sink is unnecessary
// here.
func HotPath(b *testing.B, name string, call func()) {
	b.Helper()
	b.Run(name, func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			call()
		}
	})
}

// AllocsWithin enforces an allocations-per-call budget using the
// canonical [testing.AllocsPerRun] count (100). Fails the benchmark
// when the measured allocs exceed maxAllocs. `name` is used both as
// the subtest name and as the failure label for clear CI output.
//
// AllocsPerRun runs `call` outside b.Loop, so its keep-alive
// contract does not apply. Consumers must protect non-trivial
// returns against elision — assign errors to a package-level sink
// of type error (zero-allocation interface copy), not an any-typed
// sink (boxing pollutes the alloc count).
func AllocsWithin(b *testing.B, name string, maxAllocs int, call func()) {
	b.Helper()
	b.Run(name, func(b *testing.B) {
		allocs := testing.AllocsPerRun(100, call)
		if int(allocs) > maxAllocs {
			b.Fatalf("%s: allocs %d exceeds budget %d", name, int(allocs), maxAllocs)
		}
	})
}

// LatencyWithin enforces a per-call latency budget. Uses b.Loop for
// accurate iteration scaling, then divides total elapsed time by the
// iteration count to derive per-call latency. Fails when the
// measured per-call cost exceeds maxLatency.
//
// The gate is deterministic against the benchtime — pass `-benchtime=Nx`
// for a fixed iteration count, or default `-benchtime=1s` for a
// time-bounded run that adapts to per-call cost.
func LatencyWithin(b *testing.B, name string, maxLatency time.Duration, call func()) {
	b.Helper()
	b.Run(name, func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			call()
		}
		if b.N == 0 {
			return
		}
		perCall := b.Elapsed() / time.Duration(b.N)
		if perCall > maxLatency {
			b.Fatalf("%s: latency %v/op exceeds budget %v/op", name, perCall, maxLatency)
		}
	})
}

// ConcurrentThroughput drives a multi-goroutine throughput
// measurement. b.SetParallelism(parallelism) scales workers to
// GOMAXPROCS*parallelism. The body executes inside b.RunParallel —
// the keep-alive contract of b.Loop does not extend here, so
// consumers must protect non-trivial returns against elision (assign
// errors to a package-level sink; rely on interface dispatch through
// the call to defeat elision for value returns).
func ConcurrentThroughput(b *testing.B, name string, parallelism int, call func()) {
	b.Helper()
	b.Run(name, func(b *testing.B) {
		b.SetParallelism(parallelism)
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				call()
			}
		})
	})
}

// HotPathWithBytes is a [HotPath] variant that records bytes
// processed per iteration via [testing.B.SetBytes], so the run
// reports MB/s alongside ns/op. Pass `bytesPerOp == 0` to skip the
// SetBytes call (equivalent to plain HotPath). Intended for
// throughput-meaningful shapes (Stream, BatchReader, StreamConsumer).
func HotPathWithBytes(b *testing.B, name string, bytesPerOp int64, call func()) {
	b.Helper()
	b.Run(name, func(b *testing.B) {
		if bytesPerOp > 0 {
			b.SetBytes(bytesPerOp)
		}
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			call()
		}
	})
}

// AllocsWithinWithBytes is an [AllocsWithin] variant that records
// bytes processed per iteration. The bytes report has no effect on
// the gate (allocs check is unchanged); it pairs with
// [HotPathWithBytes] for symmetric MB/s-aware output across the
// primitive set.
func AllocsWithinWithBytes(b *testing.B, name string, bytesPerOp int64, maxAllocs int, call func()) {
	b.Helper()
	b.Run(name, func(b *testing.B) {
		if bytesPerOp > 0 {
			b.SetBytes(bytesPerOp)
		}
		allocs := testing.AllocsPerRun(100, call)
		if int(allocs) > maxAllocs {
			b.Fatalf("%s: allocs %d exceeds budget %d", name, int(allocs), maxAllocs)
		}
	})
}

// LatencyWithinWithBytes is a [LatencyWithin] variant that records
// bytes processed per iteration. The latency gate is unchanged; the
// SetBytes call enables the MB/s readout.
func LatencyWithinWithBytes(b *testing.B, name string, bytesPerOp int64, maxLatency time.Duration, call func()) {
	b.Helper()
	b.Run(name, func(b *testing.B) {
		if bytesPerOp > 0 {
			b.SetBytes(bytesPerOp)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			call()
		}
		if b.N == 0 {
			return
		}
		perCall := b.Elapsed() / time.Duration(b.N)
		if perCall > maxLatency {
			b.Fatalf("%s: latency %v/op exceeds budget %v/op", name, perCall, maxLatency)
		}
	})
}

// ConcurrentThroughputWithBytes is a [ConcurrentThroughput] variant
// that records bytes processed per iteration so the parallel run
// reports MB/s aggregated across workers.
func ConcurrentThroughputWithBytes(b *testing.B, name string, bytesPerOp int64, parallelism int, call func()) {
	b.Helper()
	b.Run(name, func(b *testing.B) {
		if bytesPerOp > 0 {
			b.SetBytes(bytesPerOp)
		}
		b.SetParallelism(parallelism)
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				call()
			}
		})
	})
}

// LatencyPercentilesWithin records per-iteration latencies into a
// pre-allocated slice, sorts at the end, computes p50/p95/p99 from
// the sorted distribution, and reports each via [testing.B.ReportMetric].
// `budgets` maps percentile (0.0..1.0) → maximum acceptable per-call
// latency; any percentile that exceeds its budget fails the benchmark.
//
// Example:
//
//	bench.LatencyPercentilesWithin(b, "p99-budget",
//	    map[float64]time.Duration{
//	        0.50: 1 * time.Microsecond,
//	        0.95: 50 * time.Microsecond,
//	        0.99: 100 * time.Microsecond,
//	    }, func() {
//	        _, _ = ctx.Call(bctx, impl, key)
//	    })
//
// Pass `-benchtime=Nx` for a fixed sample size; the gate is most
// meaningful with N ≥ 1000 so the upper percentiles converge. Reports
// the measured p50/p95/p99 values regardless of whether they're
// budgeted, so consumers see the full distribution even when they
// only gate the tail.
//
// Per-iteration recording uses a manual `time.Now()` delta — slightly
// less precise than [testing.B]'s internal accounting but the only way
// to capture the distribution. Suitable for latency-sensitive shapes
// where mean-only gating would mask multi-modal behavior; not
// recommended for sub-100ns operations where the overhead of
// `time.Now()` itself dominates.
func LatencyPercentilesWithin(b *testing.B, name string, budgets map[float64]time.Duration, call func()) {
	b.Helper()
	b.Run(name, func(b *testing.B) {
		b.ReportAllocs()
		// Pre-allocate the sample buffer at b.N so per-iteration
		// recording is alloc-free. b.Loop sets b.N before the loop
		// starts, so we can size accurately.
		samples := make([]time.Duration, 0, b.N)
		b.ResetTimer()
		for b.Loop() {
			start := time.Now()
			call()
			samples = append(samples, time.Since(start))
		}
		b.StopTimer()
		if len(samples) == 0 {
			return
		}
		slices.Sort(samples)
		// Always report p50/p95/p99 so consumers see the distribution
		// even when only one percentile is budgeted.
		for _, p := range []float64{0.50, 0.95, 0.99} {
			d := percentileAt(samples, p)
			b.ReportMetric(float64(d.Nanoseconds()), fmt.Sprintf("p%d-ns/op", int(p*100)))
		}
		// Gate on every consumer-supplied budget. Sort the keys for
		// deterministic failure messages across runs.
		keys := make([]float64, 0, len(budgets))
		for p := range budgets {
			keys = append(keys, p)
		}
		sort.Float64s(keys)
		for _, p := range keys {
			d := percentileAt(samples, p)
			budget := budgets[p]
			if d > budget {
				b.Fatalf("%s: p%d latency %v exceeds budget %v",
					name, int(p*100), d, budget)
			}
		}
	})
}

// percentileAt returns the value at the requested percentile from a
// sorted slice. Uses nearest-rank interpolation: index = floor(p*n).
// Clamps to [0, len-1] so 0.0 maps to the smallest sample and 1.0
// maps to the largest.
func percentileAt(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := int(p * float64(len(sorted)))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// ReportRunningMetric exposes [testing.B.ReportMetric] for consumers
// who want to publish a custom per-iteration metric (cache hit rate,
// queue depth, GC cycles, etc.) alongside the standard ns/op +
// allocs/op output. The helper exists so consumer code in plug-in
// primitives doesn't have to reach into b directly:
//
//	StoreBenchOnGet(func(ctx bench.ReaderContext[*Store, string, Item]) {
//	    impl := ctx.Factory()
//	    var hits int64
//	    bench.HotPath(ctx.B, "with-hits", func() {
//	        if _, err := ctx.Call(ctx.B.Context(), impl, "test-key"); err == nil {
//	            hits++
//	        }
//	    })
//	    bench.ReportRunningMetric(ctx.B, "hits/op",
//	        float64(hits)/float64(ctx.B.N))
//	})
//
// The unit string is passed through to b.ReportMetric verbatim — Go's
// benchmark output convention is "<unit>/op" for per-iteration
// metrics, but any string works.
func ReportRunningMetric(b *testing.B, unit string, value float64) {
	b.Helper()
	b.ReportMetric(value, unit)
}
