// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"testing"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/rand"
	"go.thesmos.sh/testkit/stub"
)

// double mirrors a generated aggregate: one method stub, and the options a
// consumer passes at construction. DoubleBehaviour is written against
// generated code, so what exercises it here has to be wired the same way —
// every option reaching the method rather than being recorded and ignored.
type double struct {
	on *stub.MethodStub[call]
}

// newDouble applies each option to the method stub the way a generated
// constructor's loop over its Configurable slice does.
func newDouble(tb testing.TB, apply ...func(stub.Configurable)) *double {
	tb.Helper()
	d := &double{on: stub.NewMethodStub[call](tb, "Fake.Get")}
	for _, opt := range apply {
		opt(d.on)
	}
	return d
}

// instance reduces a double to what the whole-object checks drive.
func instance(d *double) stub.Instance[call] {
	return stub.Instance[call]{
		Stub: d.on,
		Call: func() {
			c := call{Key: "k"}
			d.on.SleepLatency()
			d.on.FailUnexpectedCall(c)
			d.on.Record(c)
		},
		Reset: d.on.Reset,
	}
}

// The interface-level contract is asserted once here, so a generated
// companion can bind to it rather than restate it per interface.
func TestDoubleBehaviour(t *testing.T) {
	t.Parallel()

	stub.DoubleBehaviour(t, stub.Double[call]{
		New: func(tb testing.TB) stub.Instance[call] {
			tb.Helper()
			return instance(newDouble(tb))
		},
		WithClock: func(tb testing.TB, clk clock.Clock) stub.Instance[call] {
			tb.Helper()
			return instance(newDouble(tb, func(c stub.Configurable) { c.SetClock(clk) }))
		},
		WithRandSource: func(tb testing.TB, src rand.Source) stub.Instance[call] {
			tb.Helper()
			return instance(newDouble(tb, func(c stub.Configurable) { c.SetRandSource(src) }))
		},
		BenchMode: func(tb testing.TB) stub.Instance[call] {
			tb.Helper()
			return instance(newDouble(tb, func(c stub.Configurable) { c.BenchMode() }))
		},
		Strict: func(tb testing.TB) stub.Instance[call] {
			tb.Helper()
			return instance(newDouble(tb, func(c stub.Configurable) { c.Strict() }))
		},
	})
}
