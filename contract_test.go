// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"strings"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
)

// [testkit.BenchTB] exists so the contract machinery can be driven without a
// real benchmark harness, and [testkit.FailableTB] is what satisfies it. The
// checks below are the ones a generated benchmark makes about its own ceilings,
// written against the same stand-in a generated file uses.
func TestContract(t *testing.T) {
	t.Parallel()

	t.Run("passes when no ceilings set", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(3)
		c := testkit.StartContract(b)
		for c.Loop() {
			// no-op
		}
		c.End()
		testkit.False(t, b.Failed(), "must pass with no ceilings")
		testkit.Equal(t, b.Iterations(), 3, "the loop must run the bounded number of times")
	})

	t.Run("End before Loop fatals", func(t *testing.T) {
		t.Parallel()
		// A contract whose loop never ran has measured nothing, so reporting a
		// pass would certify an empty benchmark.
		b := testkit.NewFailableTB()
		c := testkit.StartContract(b)
		c.End()
		testkit.True(t, b.Failed(), "must fatal when End called before Loop")
	})

	t.Run("latency tracking reports metrics", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(5)
		c := testkit.StartContract(b).LatencyMax(time.Second)
		for c.Loop() {
			time.Sleep(time.Microsecond)
		}
		c.End()
		testkit.False(t, b.Failed(), "must pass within generous ceiling")
		_, hasMean := b.Metric("ns/op-mean")
		_, hasP99 := b.Metric("ns/op-p99")
		testkit.True(t, hasMean, "must report mean metric")
		testkit.True(t, hasP99, "must report p99 metric")
	})

	// A hundred iterations because that is the floor below which a percentile
	// is reported and not enforced; TestContractPercentileFloor owns that
	// boundary, and this asserts only that a real violation above it fatals.
	t.Run("latency violation fatals", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(100)
		c := testkit.StartContract(b).LatencyMax(time.Nanosecond)
		for c.Loop() {
			time.Sleep(time.Microsecond)
		}
		c.End()
		testkit.True(t, b.Failed(), "must fatal on latency violation")
		testkit.True(t, strings.Contains(b.Msg(), "latency contract violated"),
			"must mention latency violation")
	})

	t.Run("alloc tracking asks the harness to report allocations", func(t *testing.T) {
		t.Parallel()
		// A contract that measures allocations without enabling their reporting
		// would gate on a number no reader of the benchmark output can see.
		b := testkit.NewFailableTB().WithIterations(10)
		c := testkit.StartContract(b).AllocsMax(1000)
		for c.Loop() {
			_ = make([]byte, 64) //nolint:gosec // force allocation for test
		}
		c.End()
		testkit.False(t, b.Failed(), "generous alloc ceiling must pass")
		testkit.True(t, b.AllocsReported(), "tracking allocations must enable their reporting")
	})

	t.Run("alloc tracking with generous ceiling passes", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(5)
		c := testkit.StartContract(b).AllocsMax(1_000_000)
		for c.Loop() {
			// no-op
		}
		c.End()
		testkit.False(t, b.Failed(), "generous ceiling must pass")
	})

	// The alloc ceiling is the whole point of the contract: a loop body that
	// allocates on every iteration must not slip past AllocsMax(0).
	//
	// Only this direction is asserted. The allocation counters are
	// process-wide, so a parallel test contributes to the delta and "a
	// non-allocating body satisfies AllocsMax(0)" is not a claim this can make.
	t.Run("alloc violation fatals", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(64)
		c := testkit.StartContract(b).AllocsMax(0)
		for c.Loop() {
			// The sink defeats escape analysis, so the allocation is real
			// and lands in MemStats.Mallocs.
			allocSink = make([]byte, 64) //nolint:gosec // forcing an allocation is the point
		}
		c.End()
		testkit.True(t, b.Failed(), "an allocating loop must violate AllocsMax(0)")
		if !strings.Contains(b.Msg(), "allocation contract violated") {
			t.Fatalf("the diagnostic must name the ceiling, got: %s", b.Msg())
		}
	})
}

// TestContractMeanMax covers the ceiling on the average, which moves
// independently of the tail: a uniform slowdown leaves p99 where it was.
func TestContractMeanMax(t *testing.T) {
	t.Parallel()

	t.Run("passes within a generous ceiling", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(5)
		c := testkit.StartContract(b).MeanMax(time.Second)
		for c.Loop() {
			time.Sleep(time.Microsecond)
		}
		c.End()
		testkit.False(t, b.Failed(), "must pass within a generous ceiling")
	})

	t.Run("fatals on violation", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(3)
		c := testkit.StartContract(b).MeanMax(time.Nanosecond)
		for c.Loop() {
			time.Sleep(time.Millisecond)
		}
		c.End()
		testkit.True(t, b.Failed(), "a millisecond body must violate a nanosecond mean")
		testkit.Contains(t, b.Msg(), "mean latency contract violated",
			"the diagnostic must name the ceiling that fired")
	})

	// The mean carries no sample floor: unlike a percentile it estimates the
	// same quantity however few iterations it saw, so three is enough to gate.
	t.Run("enforces below the percentile floor", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(3)
		c := testkit.StartContract(b).MeanMax(time.Nanosecond)
		for c.Loop() {
			time.Sleep(time.Millisecond)
		}
		c.End()
		testkit.True(t, b.Failed(),
			"the mean must gate at a sample count the percentile declines")
	})
}

// TestContractBytesMax covers the ceiling on allocation size, which moves
// independently of the count: one large allocation and many small ones are
// different regressions.
func TestContractBytesMax(t *testing.T) {
	t.Parallel()

	t.Run("reports the measured figure", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(10)
		c := testkit.StartContract(b).BytesMax(1 << 20)
		for c.Loop() {
			bytesReportSink = make([]byte, 64) //nolint:gosec // forcing an allocation is the point
		}
		c.End()
		testkit.False(t, b.Failed(), "a generous ceiling must pass")
		_, reported := b.Metric("B/op-contract")
		testkit.True(t, reported,
			"the figure a ceiling gates on must be visible in the output")
		testkit.True(t, b.AllocsReported(),
			"measuring bytes must enable allocation reporting")
	})

	// As with the alloc ceiling, only the violating direction is asserted: the
	// counters are process-wide, so a parallel test contributes to the delta.
	t.Run("fatals on violation", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(64)
		c := testkit.StartContract(b).BytesMax(8)
		for c.Loop() {
			bytesViolateSink = make([]byte, 4096) //nolint:gosec // forcing an allocation is the point
		}
		c.End()
		testkit.True(t, b.Failed(), "a 4KiB body must violate an 8-byte ceiling")
		testkit.Contains(t, b.Msg(), "byte contract violated",
			"the diagnostic must name the ceiling that fired")
	})
}

// TestContractPercentileFloor covers the rule that a percentile is reported
// but not enforced below [minPercentileSamples] iterations.
//
// The floor is a correctness guard rather than a noise one: percentile()
// indexes sorted[int(p*(n-1))], so a p99 over three samples is the median. A
// budget failing on that has failed on the shape of the sample.
func TestContractPercentileFloor(t *testing.T) {
	t.Parallel()

	t.Run("does not enforce below the floor", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(64)
		c := testkit.StartContract(b).LatencyMax(time.Nanosecond)
		for c.Loop() {
			time.Sleep(time.Microsecond)
		}
		c.End()

		testkit.False(t, b.Failed(),
			"a budget the sample count cannot support must not fail the run")
		testkit.Contains(t, strings.Join(b.Logs(), "\n"), "p99 contract not enforced",
			"and declining to enforce must say so, or it reads as a pass")
	})

	t.Run("still reports the metric below the floor", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(64)
		c := testkit.StartContract(b).LatencyMax(time.Nanosecond)
		for c.Loop() {
			time.Sleep(time.Microsecond)
		}
		c.End()

		_, hasP99 := b.Metric("ns/op-p99")
		testkit.True(t, hasP99,
			"an unenforced percentile is still the number a reader wants")
	})

	t.Run("enforces at the floor", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(100)
		c := testkit.StartContract(b).LatencyMax(time.Nanosecond)
		for c.Loop() {
			time.Sleep(time.Microsecond)
		}
		c.End()

		testkit.True(t, b.Failed(),
			"at the floor the percentile means what it says and must gate")
		testkit.Contains(t, b.Msg(), "latency contract violated",
			"the diagnostic must name the ceiling that fired")
	})
}

// allocSink and bytesSink keep loop-body allocations alive so escape analysis
// cannot stack-allocate them out of the benchmark's allocation delta.
//
// One per measuring subtest rather than one shared: they run in parallel with
// each other, and a shared sink is both a data race and a second writer inside
// a window that measures allocations process-wide.
var (
	allocSink        []byte
	bytesReportSink  []byte
	bytesViolateSink []byte
)
