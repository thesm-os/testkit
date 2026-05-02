// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"strings"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
)

// stubBench implements testkit.BenchTB for testing Contract without a real
// benchmark harness.
type stubBench struct {
	loopCount int
	looped    int
	failed    bool
	msg       string
	metrics   map[string]float64
}

func newStubBench(n int) *stubBench {
	return &stubBench{loopCount: n, metrics: make(map[string]float64)}
}

func (*stubBench) Helper()       {}
func (*stubBench) ReportAllocs() {}

func (s *stubBench) Fatal(args ...any) {
	if !s.failed {
		s.failed = true
		s.msg = args[0].(string)
	}
}

func (s *stubBench) Fatalf(format string, args ...any) {
	if !s.failed {
		s.failed = true
		s.msg = format
		if len(args) > 0 {
			// Just store format for assertion matching.
			s.msg = format
		}
	}
}

func (s *stubBench) ReportMetric(n float64, unit string) {
	s.metrics[unit] = n
}

func (s *stubBench) Loop() bool {
	if s.looped < s.loopCount {
		s.looped++
		return true
	}
	return false
}

func TestContract(t *testing.T) {
	t.Parallel()

	t.Run("passes when no ceilings set", func(t *testing.T) {
		t.Parallel()
		b := newStubBench(3)
		c := testkit.StartContract(b)
		for c.Loop() {
			// no-op
		}
		c.End()
		testkit.False(t, b.failed, "must pass with no ceilings")
	})

	t.Run("End before Loop fatals", func(t *testing.T) {
		t.Parallel()
		b := newStubBench(0)
		c := testkit.StartContract(b)
		c.End()
		testkit.True(t, b.failed, "must fatal when End called before Loop")
	})

	t.Run("latency tracking reports metrics", func(t *testing.T) {
		t.Parallel()
		b := newStubBench(5)
		c := testkit.StartContract(b).LatencyMax(time.Second)
		for c.Loop() {
			time.Sleep(time.Microsecond)
		}
		c.End()
		testkit.False(t, b.failed, "must pass within generous ceiling")
		_, hasMean := b.metrics["ns/op-mean"]
		_, hasP99 := b.metrics["ns/op-p99"]
		testkit.True(t, hasMean, "must report mean metric")
		testkit.True(t, hasP99, "must report p99 metric")
	})

	t.Run("latency violation fatals", func(t *testing.T) {
		t.Parallel()
		b := newStubBench(3)
		c := testkit.StartContract(b).LatencyMax(time.Nanosecond)
		for c.Loop() {
			time.Sleep(time.Millisecond)
		}
		c.End()
		testkit.True(t, b.failed, "must fatal on latency violation")
		testkit.True(t, strings.Contains(b.msg, "latency contract violated"),
			"must mention latency violation")
	})

	t.Run("alloc tracking runs without crash", func(t *testing.T) {
		t.Parallel()
		b := newStubBench(10)
		c := testkit.StartContract(b).AllocsMax(1000)
		for c.Loop() {
			_ = make([]byte, 64) //nolint:gosec // force allocation for test
		}
		c.End()
		// With a generous ceiling, this should pass.
		testkit.False(t, b.failed, "generous alloc ceiling must pass")
	})

	t.Run("alloc tracking with generous ceiling passes", func(t *testing.T) {
		t.Parallel()
		b := newStubBench(5)
		c := testkit.StartContract(b).AllocsMax(1_000_000)
		for c.Loop() {
			// no-op
		}
		c.End()
		testkit.False(t, b.failed, "generous ceiling must pass")
	})
}
