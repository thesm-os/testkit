// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
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
