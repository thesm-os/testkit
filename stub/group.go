// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"testing"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/rand"
)

// Group fans a double-wide setting across every method stub of one
// generated double. Each double used to restate the loops — clock,
// rand source, bench mode, strict, reset — and the fuzz-guarded cleanup
// that verifies call-count expectations; the group is their one home,
// and a generated constructor is one NewGroup call plus one Bind.
type Group struct {
	members []Configurable
}

// NewGroup collects a double's method stubs.
func NewGroup(members ...Configurable) *Group {
	return &Group{members: members}
}

// Bind registers the end-of-test verification of every member's
// call-count expectations. A nil tb skips it, which is what benchmarks
// and non-test callers want — and why the guard runs before any tb
// method, Helper included. A fuzz target is skipped too: it reruns
// its body many times over and registers a cleanup per run, so
// verifying there would report against the wrong iteration.
//
//nolint:thelper
func (g *Group) Bind(tb testing.TB) {
	if tb == nil {
		return
	}
	if _, isFuzz := tb.(*testing.F); isFuzz {
		return
	}
	tb.Cleanup(func() {
		for _, m := range g.members {
			m.Verify()
		}
	})
}

// StrictAll makes every unconfigured method fail the test rather than
// return its zero value.
func (g *Group) StrictAll() {
	for _, m := range g.members {
		m.Strict()
	}
}

// SetClock drives latency and time-windowed faults from clk rather than
// wall time, on every member.
func (g *Group) SetClock(clk clock.Clock) {
	for _, m := range g.members {
		m.SetClock(clk)
	}
}

// SetRandSource makes probabilistic fault injection reproducible, on
// every member.
func (g *Group) SetRandSource(src rand.Source) {
	for _, m := range g.members {
		m.SetRandSource(src)
	}
}

// BenchMode disables call recording on every member — the recording's
// allocations are what a benchmark would otherwise be measuring.
func (g *Group) BenchMode() {
	for _, m := range g.members {
		m.BenchMode()
	}
}

// Reset clears recorded calls, fault counters, and call-count
// expectations on every member, leaving Func and Returns configuration
// in place.
func (g *Group) Reset() {
	for _, m := range g.members {
		m.Reset()
	}
}
