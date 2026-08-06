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

	t.Run("latency violation fatals", func(t *testing.T) {
		t.Parallel()
		b := testkit.NewFailableTB().WithIterations(3)
		c := testkit.StartContract(b).LatencyMax(time.Nanosecond)
		for c.Loop() {
			time.Sleep(time.Millisecond)
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
	// Only this direction is asserted. runtime.MemStats counts allocations
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

// allocSink keeps loop-body allocations alive so escape analysis cannot
// stack-allocate them out of the benchmark's MemStats delta.
var allocSink []byte
