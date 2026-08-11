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
	Logf(format string, args ...any)
	ReportAllocs()
	ReportMetric(n float64, unit string)
	Loop() bool
}

// Contract provides scope-based allocation and latency assertions for
// benchmarks. It is a drop-in replacement for the b.N loop pattern that
// additionally enforces ceilings on allocations, bytes and latency.
//
//	func BenchmarkStore_Get(b *testing.B) {
//	    s := setup()
//	    c := testkit.StartContract(b).AllocsMax(0).LatencyMax(5 * time.Microsecond)
//	    for c.Loop() {
//	        _ = s.Get(b.Context(), key)
//	    }
//	    c.End()
//	}
//
// Four ceilings, each opt-in and each naming one regression: a ceiling nobody
// sets is not enforced. [Contract.AllocsMax] and [Contract.BytesMax] bound
// allocation count and size, which move independently — one large allocation
// and many small ones are different defects. [Contract.LatencyMax] and
// [Contract.MeanMax] bound the tail and the average, which also move
// independently: a p99 that moved alone is a new tail, a mean that moved alone
// is a uniform slowdown, and a budget naming one cannot report the other.
//
// # Allocation
//
// The allocation figures come from [runtime.ReadMemStats], which is what
// [testing.B] derives its own `allocs/op` and `B/op` from — so a ceiling here
// gates the number `-benchmem` already puts in front of the reader.
//
// Its stop-the-world is the mechanism rather than a cost to be avoided: it is
// what flushes the per-P allocation caches so the figure is exact and current.
// The [runtime/metrics] counters `/gc/heap/allocs:objects` and
// `/gc/heap/allocs:bytes` are not a substitute — they read an aggregate that
// lags until something flushes those caches, and a thousand tiny allocations
// can move [runtime.MemStats.Mallocs] by a thousand while moving the metrics by
// nothing at all. A contract reading them would report zero allocations for a
// body that allocates.
type Contract struct {
	tb BenchTB

	maxAllocs  uint64
	maxBytes   uint64
	maxLatency time.Duration
	maxMean    time.Duration

	trackAllocs  bool
	trackBytes   bool
	trackLatency bool
	trackMean    bool

	before    runtime.MemStats
	baselined bool

	durations []time.Duration
	iterStart time.Time
	started   bool
	iter      uint64
}

// minPercentileSamples is the iteration count below which a percentile is
// reported but not enforced.
//
// Not a noise threshold but a correctness one: [percentile] indexes
// `sorted[int(p*(n-1))]`, so a p99 over three samples is `sorted[1]` — the
// median — and over two it is `sorted[0]`, the minimum. Below this floor the
// number reported as p99 is a different statistic wearing its name, and gating
// on it fails a run for the shape of its sample rather than the speed of its
// code.
//
// The floor is the percentile's alone. A mean is an unbiased estimator at any
// count, and allocation figures are per-operation averages; each measures what
// it claims however few iterations it saw.
const minPercentileSamples = 100

// StartContract begins a new benchmark contract on tb. Chain [Contract.AllocsMax],
// [Contract.BytesMax], [Contract.LatencyMax] and [Contract.MeanMax] to set
// ceilings before calling [Contract.Loop].
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

// BytesMax sets the maximum bytes allocated per iteration.
//
// The figure is the one `go test -benchmem` prints as `B/op`: [testing.B]
// derives that from the same cumulative counter, so a ceiling here gates the
// number a reader already has in front of them.
func (c *Contract) BytesMax(n uint64) *Contract {
	c.maxBytes = n
	c.trackBytes = true
	return c
}

// LatencyMax sets the maximum p99 latency per iteration.
//
// Enforced only from [minPercentileSamples] iterations upward; below that the
// percentile is reported with a note and the run is not failed.
func (c *Contract) LatencyMax(d time.Duration) *Contract {
	c.maxLatency = d
	c.trackLatency = true
	return c
}

// MeanMax sets the maximum mean latency per iteration.
func (c *Contract) MeanMax(d time.Duration) *Contract {
	c.maxMean = d
	c.trackMean = true
	return c
}

// wantsMem and wantsTime report whether any ceiling needs the corresponding
// measurement collected. Two ceilings share each source, and collection is
// owed when either is declared.
func (c *Contract) wantsMem() bool  { return c.trackAllocs || c.trackBytes }
func (c *Contract) wantsTime() bool { return c.trackLatency || c.trackMean }

// Loop is a drop-in replacement for [testing.B.Loop]. It delegates to the
// underlying BenchTB.Loop() and captures per-iteration timing when a latency
// ceiling is declared.
func (c *Contract) Loop() bool {
	if c.started && c.wantsTime() {
		c.durations = append(c.durations, time.Since(c.iterStart))
	}

	if !c.started {
		c.started = true
		if c.wantsMem() {
			c.tb.ReportAllocs()
		}
	}

	ok := c.tb.Loop()
	if ok {
		c.iter++
		// The allocation baseline is taken after the first delegated Loop, not
		// before it: that call runs [testing.B.ResetTimer], which allocates the
		// map holding custom metrics. Reading earlier would charge the
		// framework's own allocation to the subject's budget, which under
		// `-benchtime=1x` is the whole measurement.
		if !c.baselined && c.wantsMem() {
			c.baselined = true
			runtime.ReadMemStats(&c.before)
		}
		// Set last, so neither the baseline read nor the bookkeeping above
		// lands inside the first iteration's duration.
		if c.wantsTime() {
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

	c.endMemory()
	c.endLatency()
}

// endMemory enforces the allocation ceilings.
//
// Guarded on baselined as well as on the iteration count: a loop that never
// ran has no baseline to subtract, and a delta against the zero value would
// report the process's entire allocation history as one operation's cost.
func (c *Contract) endMemory() {
	if !c.wantsMem() || !c.baselined || c.iter == 0 {
		return
	}
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	if c.trackBytes {
		bytesPerOp := (after.TotalAlloc - c.before.TotalAlloc) / c.iter
		c.tb.ReportMetric(float64(bytesPerOp), "B/op-contract")
		if bytesPerOp > c.maxBytes {
			c.tb.Fatalf(
				"byte contract violated: %d B/op, max %d",
				bytesPerOp, c.maxBytes,
			)
		}
	}

	if c.trackAllocs {
		allocsPerOp := (after.Mallocs - c.before.Mallocs) / c.iter
		if allocsPerOp > c.maxAllocs {
			c.tb.Fatalf(
				"allocation contract violated: %d allocs/op, max %d",
				allocsPerOp, c.maxAllocs,
			)
		}
	}
}

// endLatency reports the timing metrics and enforces the ceilings that have
// enough samples to stand on.
func (c *Contract) endLatency() {
	if !c.wantsTime() || len(c.durations) == 0 {
		return
	}

	p99 := percentile(c.durations, 0.99)
	mean := meanDuration(c.durations)
	c.tb.ReportMetric(float64(mean.Nanoseconds()), "ns/op-mean")
	c.tb.ReportMetric(float64(p99.Nanoseconds()), "ns/op-p99")

	if c.trackMean && mean > c.maxMean {
		c.tb.Fatalf(
			"mean latency contract violated: mean %v, max %v",
			mean, c.maxMean,
		)
	}

	switch {
	case !c.trackLatency:
	case len(c.durations) < minPercentileSamples:
		c.tb.Logf(
			"p99 contract not enforced: %d iterations, %d required — "+
				"raise -benchtime to gate this budget",
			len(c.durations), minPercentileSamples,
		)
	case p99 > c.maxLatency:
		c.tb.Fatalf(
			"latency contract violated: p99 %v, max %v",
			p99, c.maxLatency,
		)
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
