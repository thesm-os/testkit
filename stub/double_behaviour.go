// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/rand"
)

// latencyWindow is the latency the clock-driven checks configure. Any
// duration works — the point is that a virtual clock crosses it instantly —
// but a value far larger than any plausible real delay makes a check that
// accidentally slept on wall time obvious rather than merely slow.
const latencyWindow = 5 * time.Second

// realTimeBudget is how long a virtual-clock check may take on the wall
// before it has plainly slept for real. Generous enough that a loaded machine
// does not fail it, small enough that a five-second real sleep cannot pass.
const realTimeBudget = 500 * time.Millisecond

// Instance is one constructed double, reduced to what the whole-object
// checks need: a method to drive, a way to drive it, and a way to reset.
type Instance[C any] struct {
	// Stub is the method stub the checks configure and inspect. Any method
	// will do — what is under test is that a setting reached the double at
	// all, and a generated companion picks the first.
	Stub *MethodStub[C]

	// Call invokes that method with zero-valued arguments.
	Call func()

	// Reset clears the double, standing in for the generated ResetCalls.
	Reset func()
}

// Double describes how to construct a generated double under each of the
// options whose effect is signature-independent.
//
// Where [Subject] describes one method, this describes the object. Options
// are separate constructors rather than values threaded through one, because
// a generated `<Iface>StubOption` cannot be named here without naming the
// double — and because several are read during construction rather than
// applied after it.
type Double[C any] struct {
	// New builds a plain double.
	New func(tb testing.TB) Instance[C]

	// WithClock builds one whose latency and time-windowed faults are driven
	// by clk rather than by wall time.
	WithClock func(tb testing.TB, clk clock.Clock) Instance[C]

	// WithRandSource builds one whose probabilistic faults draw from src.
	WithRandSource func(tb testing.TB, src rand.Source) Instance[C]

	// BenchMode builds one with call recording disabled.
	BenchMode func(tb testing.TB) Instance[C]

	// Strict builds one whose unconfigured methods fail the test.
	Strict func(tb testing.TB) Instance[C]
}

// DoubleBehaviour runs the contract a generated double owes as a whole,
// independent of any method's signature.
//
// Every check here concerns a setting supposed to reach the double rather
// than one method of it, or a property of the recording itself. A generated
// companion restating them would assert the same thing once per interface,
// and these are the assertions most likely to be subtly wrong — a clock that
// never reached a method, a latency slept after the call was already
// recorded.
//
// # Hazards
//
// The latency checks drive a virtual clock from another goroutine and would
// block rather than fail if the double slept on wall time, so each is bounded
// by a real-time budget. A failure there means latency is not clock-driven,
// not that the machine is slow.
func DoubleBehaviour[C any](t *testing.T, d Double[C]) {
	t.Helper()

	t.Run("stops recording in bench mode", func(t *testing.T) {
		t.Parallel()
		// The accumulating log is what a benchmark would otherwise be
		// measuring, so an option that dispatches correctly but keeps
		// recording defeats the purpose without failing anything.
		i := d.BenchMode(t)
		i.Call()
		testkit.Equal(t, i.Stub.CallCount(), 0, "bench mode must stop the call log growing")
	})

	t.Run("refuses an unconfigured call when built strict", func(t *testing.T) {
		t.Parallel()
		// Strict is reachable per method, but a consumer sets it once for the
		// whole double — and an option that reached only some methods would
		// leave the rest silently permissive.
		f := testkit.NewFailableTB()
		i := d.Strict(f)
		i.Call()
		assertFailed(t, f, "a strict double must refuse an unconfigured call")
	})

	t.Run("clears timestamps on reset", func(t *testing.T) {
		t.Parallel()
		i := d.New(t)
		i.Call()
		testkit.Len(t, i.Stub.Timestamped(), 1, "a call must be timestamped")
		i.Reset()
		testkit.Len(t, i.Stub.Timestamped(), 0, "reset must clear timestamps with the log")
	})

	t.Run("draws probabilistic faults from the injected source", func(t *testing.T) {
		t.Parallel()
		// A fixed source at 0.5 fires a fault configured above it and not one
		// configured below, which is observable only if the source actually
		// reached the method. A source that never landed leaves probabilistic
		// faults unreproducible rather than visibly broken.
		i := d.WithRandSource(t, rand.FixedRandSource(0.5))
		i.Stub.FaultsWithProbability(0.9, testkit.TestError("double-behaviour"))
		fired, _ := i.Stub.ShouldFaultFor(*new(C))
		testkit.True(t, fired, "a fault above the source's draw must fire")
	})

	doubleLatency(t, d)
}

// doubleLatency runs the clock-driven checks, which are the ones most worth
// having: latency that silently used wall time would make every consumer's
// suite slow rather than wrong, and nothing else would report it.
func doubleLatency[C any](t *testing.T, d Double[C]) {
	t.Helper()

	t.Run("sleeps latency against the injected clock", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(time.Unix(0, 0))
		i := d.WithClock(t, clk)
		i.Stub.Latency(latencyWindow)

		done := make(chan struct{})
		go func() {
			i.Call()
			close(done)
		}()

		clk.AwaitWaiters(1)
		start := time.Now()
		clk.Advance(latencyWindow + time.Second)
		<-done

		testkit.True(t, time.Since(start) < realTimeBudget,
			"latency must come from the injected clock rather than wall time")
	})

	t.Run("records a call only once its latency has elapsed", func(t *testing.T) {
		t.Parallel()
		// A call recorded before its latency elapses reports as complete
		// while still in flight, which is exactly the window a concurrency
		// test is trying to observe.
		clk := clock.NewTestClock(time.Unix(0, 0))
		i := d.WithClock(t, clk)
		i.Stub.Latency(latencyWindow)

		done := make(chan struct{})
		go func() {
			i.Call()
			close(done)
		}()

		clk.AwaitWaiters(1)
		testkit.Equal(t, i.Stub.CallCount(), 0, "a call inside its latency window is not yet recorded")
		clk.Advance(latencyWindow + time.Second)
		<-done
		testkit.Equal(t, i.Stub.CallCount(), 1, "the call is recorded once latency elapses")
	})

	t.Run("stops a time-windowed fault once its window closes", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(time.Unix(0, 0))
		i := d.WithClock(t, clk)
		i.Stub.FaultsFor(latencyWindow, testkit.TestError("double-behaviour"))

		fired, _ := i.Stub.ShouldFaultFor(*new(C))
		testkit.True(t, fired, "a fault inside its window fires")

		clk.Advance(latencyWindow + time.Second)
		after, _ := i.Stub.ShouldFaultFor(*new(C))
		testkit.False(t, after, "a fault past its window stops firing")
	})
}
