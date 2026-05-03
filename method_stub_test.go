// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
)

type sampleCall struct {
	Arg    string
	Result int
	Err    error
}

func TestMethodStub(t *testing.T) {
	t.Parallel()

	t.Run("records calls via embedded Recorder", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Record(sampleCall{Arg: "a", Result: 1})
		stub.Record(sampleCall{Arg: "b", Result: 2})

		testkit.Equal(t, stub.CallCount(), 2, "must record calls")
		calls := stub.Calls()
		testkit.Equal(t, calls[0].Arg, "a", "first call arg")
		testkit.Equal(t, calls[1].Result, 2, "second call result")
	})

	t.Run("assertion helpers delegate to Recorder", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Record(sampleCall{Arg: "only"})

		call := stub.AssertCalledOnce(t, "must be called once")
		testkit.Equal(t, call.Arg, "only", "must return the call")
	})

	t.Run("Filter delegates to Recorder", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Record(sampleCall{Arg: "a"})
		stub.Record(sampleCall{Arg: "b"})
		stub.Record(sampleCall{Arg: "a"})

		filtered := stub.Filter(func(c sampleCall) bool { return c.Arg == "a" })
		testkit.Len(t, filtered, 2, "must filter correctly")
	})
}

func TestMethodStubFaults(t *testing.T) {
	t.Parallel()

	t.Run("ShouldFault returns false when not configured", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		fired, err := stub.ShouldFaultFor(sampleCall{})
		testkit.False(t, fired, "must not fire without configuration")
		testkit.NoError(t, err, "err must be nil")
	})

	t.Run("ShouldFault fires on configured interval", func(t *testing.T) {
		t.Parallel()
		errBoom := testkit.TestError("boom")
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Faults(errBoom, 2) // fires every 2nd call

		f1, _ := stub.ShouldFaultFor(sampleCall{})  // call 1: no
		f2, e2 := stub.ShouldFaultFor(sampleCall{}) // call 2: yes
		f3, _ := stub.ShouldFaultFor(sampleCall{})  // call 3: no
		f4, e4 := stub.ShouldFaultFor(sampleCall{}) // call 4: yes

		testkit.False(t, f1, "call 1 must not fault")
		testkit.True(t, f2, "call 2 must fault")
		testkit.ErrorIs(t, e2, errBoom, "call 2 fault error")
		testkit.False(t, f3, "call 3 must not fault")
		testkit.True(t, f4, "call 4 must fault")
		testkit.ErrorIs(t, e4, errBoom, "call 4 fault error")
	})

	t.Run("Reset clears fault counter", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Faults(testkit.TestError("x"), 2)
		_, _ = stub.ShouldFaultFor(sampleCall{}) // call 1 — advance counter
		stub.Record(sampleCall{Arg: "a"})

		stub.Reset()

		testkit.Equal(t, stub.CallCount(), 0, "recordings must be cleared")
		f1, _ := stub.ShouldFaultFor(sampleCall{})
		testkit.False(t, f1, "fault counter must be reset")
	})
}

func TestMethodStubStrict(t *testing.T) {
	t.Parallel()

	t.Run("IsStrict reports mode", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		testkit.False(t, stub.IsStrict(), "default is lenient")
		stub.Strict()
		testkit.True(t, stub.IsStrict(), "must be strict after Strict()")
	})

	t.Run("FailUnexpectedCall does nothing in lenient mode", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.FailUnexpectedCall(sampleCall{Arg: "test"})
	})

	t.Run("FailUnexpectedCall fatals in strict mode with call details", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.Strict()

		stub.FailUnexpectedCall(sampleCall{Arg: "test-arg"})
		testkit.True(t, f.Failed(), "must fatal in strict mode")
		testkit.Assert(t, f.Msg()).
			Contains("Svc.Do", "must include method name").
			Contains("unexpected call", "must describe the failure").
			Contains("test-arg", "must include call details")
	})
}

func TestMethodStubVerify(t *testing.T) {
	t.Parallel()

	t.Run("Times passes when count matches", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.Times(2)
		stub.Record(sampleCall{})
		stub.Record(sampleCall{})

		stub.Verify()
		testkit.False(t, f.Failed(), "must pass when count matches")
	})

	t.Run("Times fails when count does not match", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.Times(3)
		stub.Record(sampleCall{})

		stub.Verify()
		testkit.True(t, f.Failed(), "must fail when count does not match")
		testkit.Assert(t, f.Msg()).
			Contains("Svc.Do", "must include method name").
			Contains("expected 3", "must include expected count").
			Contains("got 1", "must include actual count")
	})

	t.Run("TimesAtLeast passes when count sufficient", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.TimesAtLeast(1)
		stub.Record(sampleCall{})
		stub.Record(sampleCall{})

		stub.Verify()
		testkit.False(t, f.Failed(), "must pass when count >= minimum")
	})

	t.Run("TimesAtLeast fails when count insufficient", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.TimesAtLeast(5)
		stub.Record(sampleCall{})

		stub.Verify()
		testkit.True(t, f.Failed(), "must fail when count < minimum")
		testkit.Assert(t, f.Msg()).Contains("at least 5", "must include minimum")
	})

	t.Run("Verify does nothing without tb", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Times(99)
		stub.Verify() // should not panic
	})

	t.Run("Reset clears expectations", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.Times(5)
		stub.Reset()

		stub.Verify()
		testkit.False(t, f.Failed(), "reset must clear Times expectation")
	})
}

func TestMethodStubAdvancedFaults(t *testing.T) {
	t.Parallel()

	t.Run("FaultsWhen fires on matching calls", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		errTarget := testkit.TestError("targeted")
		stub.FaultsWhen(func(c sampleCall) bool { return c.Arg == "target" }, errTarget, 1)

		fired, err := stub.ShouldFaultFor(sampleCall{Arg: "target"})
		testkit.True(t, fired, "must fire for matching call")
		testkit.ErrorIs(t, err, errTarget, "must return targeted error")
	})

	t.Run("FaultsWhen skips non-matching calls", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.FaultsWhen(func(c sampleCall) bool { return c.Arg == "target" }, testkit.TestError("x"), 1)

		fired, _ := stub.ShouldFaultFor(sampleCall{Arg: "other"})
		testkit.False(t, fired, "must not fire for non-matching call")
	})

	t.Run("FaultsWithProbability uses configured RandSource", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.WithRandSource(testkit.FixedRandSource(0.3))
		errBoom := testkit.TestError("boom")
		stub.FaultsWithProbability(0.5, errBoom) // 0.3 < 0.5 → fires

		fired, err := stub.ShouldFaultFor(sampleCall{})
		testkit.True(t, fired, "must fire with rand < p")
		testkit.ErrorIs(t, err, errBoom, "must return configured error")
	})

	t.Run("FaultsWithProbability uses default source when none set", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.FaultsWithProbability(0.0, testkit.TestError("x")) // p=0 never fires

		fired, _ := stub.ShouldFaultFor(sampleCall{})
		testkit.False(t, fired, "p=0 must never fire")
	})

	t.Run("FaultsFor fires within window", func(t *testing.T) {
		t.Parallel()
		origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		clk := testkit.NewTestClock(origin)
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.WithClock(clk)
		errDown := testkit.TestError("down")
		stub.FaultsFor(5*time.Second, errDown)

		fired, _ := stub.ShouldFaultFor(sampleCall{})
		testkit.True(t, fired, "must fire within window")

		clk.Advance(6 * time.Second)
		fired, _ = stub.ShouldFaultFor(sampleCall{})
		testkit.False(t, fired, "must not fire after window")
	})

	t.Run("FaultsUntil fires before deadline", func(t *testing.T) {
		t.Parallel()
		origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		clk := testkit.NewTestClock(origin)
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.WithClock(clk)
		deadline := origin.Add(10 * time.Second)
		stub.FaultsUntil(deadline, testkit.TestError("down"))

		fired, _ := stub.ShouldFaultFor(sampleCall{})
		testkit.True(t, fired, "must fire before deadline")

		clk.Advance(11 * time.Second)
		fired, _ = stub.ShouldFaultFor(sampleCall{})
		testkit.False(t, fired, "must not fire after deadline")
	})

	t.Run("ShouldFaultFor returns false with no fault", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		fired, err := stub.ShouldFaultFor(sampleCall{})
		testkit.False(t, fired, "must not fire with no fault")
		testkit.NoError(t, err, "must return nil error")
	})
}

func TestMethodStubLatency(t *testing.T) {
	t.Parallel()

	t.Run("SleepLatency is no-op without config", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.SleepLatency() // should return immediately
	})

	t.Run("SleepLatency uses configured clock", func(t *testing.T) {
		t.Parallel()
		origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		clk := testkit.NewTestClock(origin)
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.WithClock(clk)
		stub.Latency(5 * time.Second)

		done := make(chan struct{})
		go func() {
			stub.SleepLatency()
			close(done)
		}()

		time.Sleep(10 * time.Millisecond)
		start := time.Now()
		clk.Advance(6 * time.Second)
		<-done
		elapsed := time.Since(start)
		testkit.True(t, elapsed < 100*time.Millisecond,
			"SleepLatency must use virtual clock, not real time")
	})

	t.Run("SleepLatency uses real clock when none configured", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Latency(5 * time.Millisecond)
		start := time.Now()
		stub.SleepLatency()
		testkit.True(t, time.Since(start) >= 5*time.Millisecond,
			"must use real time when no clock configured")
	})

	t.Run("Latency(0) disables sleep", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Latency(5 * time.Second)
		stub.Latency(0)     // disable
		stub.SleepLatency() // should return immediately
	})

	t.Run("Clock returns configured clock", func(t *testing.T) {
		t.Parallel()
		clk := testkit.NewTestClock(time.Unix(0, 0))
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		testkit.True(t, stub.Clock() == nil, "default must be nil")
		stub.WithClock(clk)
		testkit.True(t, stub.Clock() == clk, "must return configured clock")
	})
}

func TestMethodStubName(t *testing.T) {
	t.Parallel()

	t.Run("Name returns configured name", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Store.Get")
		testkit.Equal(t, stub.Name(), "Store.Get", "must return name")
	})

	t.Run("TB returns configured TB", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		testkit.True(t, stub.TB() == f, "must return configured TB")
	})

	t.Run("TB returns nil when not configured", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		testkit.True(t, stub.TB() == nil, "must return nil")
	})
}
