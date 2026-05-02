// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"runtime"
	"slices"
	"time"
)

// BenchTB is the subset of [testing.B] that [Contract] requires. Defining it
// as an interface allows testing the contract machinery itself without a real
// benchmark harness.
type BenchTB interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	ReportAllocs()
	ReportMetric(n float64, unit string)
	Loop() bool
}

// Contract provides scope-based allocation and latency assertions for
// benchmarks. It is a drop-in replacement for the b.N loop pattern that
// additionally enforces ceilings on allocations and p99 latency.
//
//	func BenchmarkStore_Get(b *testing.B) {
//	    s := setup()
//	    c := testkit.StartContract(b).AllocsMax(0).LatencyMax(5 * time.Microsecond)
//	    for c.Loop() {
//	        _ = s.Get(b.Context(), key)
//	    }
//	    c.End()
//	}
type Contract struct {
	tb           BenchTB
	maxAllocs    uint64
	maxLatency   time.Duration
	trackAllocs  bool
	trackLatency bool
	before       runtime.MemStats
	durations    []time.Duration
	iterStart    time.Time
	started      bool
	iter         uint64
}

// StartContract begins a new benchmark contract on tb. Chain [Contract.AllocsMax]
// and [Contract.LatencyMax] to set ceilings before calling [Contract.Loop].
func StartContract(tb BenchTB) *Contract {
	tb.Helper()
	return &Contract{tb: tb}
}

// AllocsMax sets the maximum number of heap allocations per iteration.
// Use 0 for zero-allocation contracts.
func (c *Contract) AllocsMax(n uint64) *Contract {
	c.maxAllocs = n
	c.trackAllocs = true
	return c
}

// LatencyMax sets the maximum p99 latency per iteration.
func (c *Contract) LatencyMax(d time.Duration) *Contract {
	c.maxLatency = d
	c.trackLatency = true
	return c
}

// Loop is a drop-in replacement for [testing.B.Loop]. It delegates to the
// underlying BenchTB.Loop() and captures per-iteration timing when latency
// tracking is enabled.
func (c *Contract) Loop() bool {
	if c.started && c.trackLatency {
		c.durations = append(c.durations, time.Since(c.iterStart))
	}

	if !c.started {
		c.started = true
		if c.trackAllocs {
			c.tb.ReportAllocs()
			runtime.ReadMemStats(&c.before)
		}
	}

	ok := c.tb.Loop()
	if ok {
		c.iter++
		if c.trackLatency {
			c.iterStart = time.Now()
		}
	}
	return ok
}

// End reports metrics and asserts ceilings. Call this after the Loop exits.
// It calls tb.Fatalf if any ceiling is exceeded.
func (c *Contract) End() {
	c.tb.Helper()
	if !c.started {
		c.tb.Fatal("Contract.End called before Loop")
		return
	}

	if c.trackAllocs {
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		totalAllocs := after.Mallocs - c.before.Mallocs
		if c.iter > 0 {
			allocsPerOp := totalAllocs / c.iter
			if allocsPerOp > c.maxAllocs {
				c.tb.Fatalf(
					"allocation contract violated: %d allocs/op, max %d",
					allocsPerOp, c.maxAllocs,
				)
			}
		}
	}

	if c.trackLatency && len(c.durations) > 0 {
		p99 := percentile(c.durations, 0.99)
		mean := meanDuration(c.durations)
		c.tb.ReportMetric(float64(mean.Nanoseconds()), "ns/op-mean")
		c.tb.ReportMetric(float64(p99.Nanoseconds()), "ns/op-p99")
		if p99 > c.maxLatency {
			c.tb.Fatalf(
				"latency contract violated: p99 %v, max %v",
				p99, c.maxLatency,
			)
		}
	}
}

func percentile(ds []time.Duration, p float64) time.Duration {
	sorted := make([]time.Duration, len(ds))
	copy(sorted, ds)
	slices.Sort(sorted)
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func meanDuration(ds []time.Duration) time.Duration {
	var total time.Duration
	for _, d := range ds {
		total += d
	}
	return total / time.Duration(len(ds))
}
