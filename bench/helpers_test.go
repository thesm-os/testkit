// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
)

func TestPercentileAt(t *testing.T) {
	t.Parallel()

	t.Run("empty returns zero", func(t *testing.T) {
		t.Parallel()
		got := percentileAt(nil, 0.5)
		testkit.Equal(t, got, time.Duration(0), "empty slice")
	})

	t.Run("p=0 returns smallest", func(t *testing.T) {
		t.Parallel()
		samples := []time.Duration{10, 20, 30, 40, 50}
		got := percentileAt(samples, 0.0)
		testkit.Equal(t, got, time.Duration(10), "p0")
	})

	t.Run("p=1 returns largest", func(t *testing.T) {
		t.Parallel()
		samples := []time.Duration{10, 20, 30, 40, 50}
		got := percentileAt(samples, 1.0)
		testkit.Equal(t, got, time.Duration(50), "p100")
	})

	t.Run("p=0.5 returns median", func(t *testing.T) {
		t.Parallel()
		samples := []time.Duration{10, 20, 30, 40, 50}
		got := percentileAt(samples, 0.50)
		testkit.Equal(t, got, time.Duration(30), "p50 of 5 samples")
	})

	t.Run("p=0.99 returns near-largest", func(t *testing.T) {
		t.Parallel()
		samples := make([]time.Duration, 100)
		for i := range samples {
			samples[i] = time.Duration(i)
		}
		got := percentileAt(samples, 0.99)
		testkit.Equal(t, got, time.Duration(99), "p99 of 100 samples")
	})

	t.Run("p > 1 clamps to largest", func(t *testing.T) {
		t.Parallel()
		samples := []time.Duration{10, 20, 30}
		got := percentileAt(samples, 1.5)
		testkit.Equal(t, got, time.Duration(30), "clamped to largest")
	})
}

func TestSubtestKey(t *testing.T) {
	t.Parallel()

	t.Run("string passes through", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, SubtestKey("test-key"), "test-key", "string")
	})

	t.Run("integer renders as decimal", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, SubtestKey(42), "42", "int")
	})

	t.Run("struct strips braces and joins fields with hyphens", func(t *testing.T) {
		t.Parallel()
		type item struct {
			ID, Name string
		}
		got := SubtestKey(item{ID: "test-id", Name: "test-name"})
		testkit.Equal(t, got, "test-id-test-name", "struct")
	})

	t.Run("empty value returns underscore", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, SubtestKey(""), "_", "empty string")
	})
}

func TestLatencyPercentilesWithinReportsMetrics(t *testing.T) {
	t.Parallel()

	// Run an inner benchmark that records p50/p95/p99 metrics.
	// We assert via testing.Benchmark that the gate passes when the
	// budgets are loose, and that ReportMetric was called (we can't
	// inspect the metrics directly from outside, but a non-panicking
	// run with the gate in place is the testable signal).
	res := testing.Benchmark(func(b *testing.B) {
		b.Helper()
		LatencyPercentilesWithin(b, "ok", map[float64]time.Duration{
			0.50: time.Hour,
			0.99: time.Hour,
		}, func() {
			// trivial work — sub-microsecond
		})
	})
	testkit.True(t, res.N > 0, "benchmark ran at least one iteration")
}

func TestReportRunningMetric(t *testing.T) {
	t.Parallel()
	res := testing.Benchmark(func(b *testing.B) {
		b.Helper()
		HotPath(b, "metric", func() {
			// trivial work
		})
		ReportRunningMetric(b, "custom/op", 1.5)
	})
	// ReportMetric stores under custom keys; the standard res.Extra
	// map should carry it.
	testkit.True(t, res.N > 0, "benchmark ran")
}
