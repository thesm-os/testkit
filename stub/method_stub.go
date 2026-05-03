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

// MethodStub is a generic per-method test double that combines recording,
// fault injection, strict mode, and call count expectations. Generated
// stub code embeds one MethodStub per interface method and adds type-specific
// dispatch (Func, Returns).
//
//	type GetStub struct {
//	    *testkit.MethodStub[GetCall]
//	    fn       func(context.Context, string) (Item, error)
//	    fallback *getReturn
//	}
type MethodStub[C any] struct {
	*Recorder[C]

	fault   testkit.Fault[C]
	clock   clock.Clock
	rand    rand.Source
	latency time.Duration
	tb      testing.TB
	name    string // "Store.Get" — for error messages
	strict  bool
	times   *int
	atLeast *int
}

// NewMethodStub creates a [MethodStub] for the named method. Pass tb for
// auto-verification via [testing.TB.Cleanup]; pass nil for a pure stub
// without test integration.
//
//nolint:thelper // constructor, not a test helper — tb may be nil
func NewMethodStub[C any](tb testing.TB, name string) *MethodStub[C] {
	return &MethodStub[C]{
		Recorder: NewRecorder[C](),
		tb:       tb,
		name:     name,
	}
}

// Strict enables strict mode. In strict mode, [MethodStub.FailUnexpected]
// fatals the test when called — generated stubs call FailUnexpected when
// no behavior (func, returns, matcher) is configured.
func (s *MethodStub[C]) Strict() {
	s.strict = true
}

// IsStrict reports whether strict mode is enabled.
func (s *MethodStub[C]) IsStrict() bool {
	return s.strict
}

// Faults configures counter-based fault injection on this method. The fault
// fires on every nth call. Returns the receiver for chaining.
//
// For more advanced fault patterns, see [MethodStub.SetFault] which accepts
// any [Fault] strategy.
func (s *MethodStub[C]) Faults(err error, failEveryN int) *MethodStub[C] {
	s.fault = testkit.NewCountedFault[C](err, failEveryN)
	return s
}

// SetFault configures an arbitrary fault strategy on this method. Use this
// for advanced patterns (predicate-based, probabilistic, time-windowed, or
// composed faults). For simple counter-based faults, prefer [MethodStub.Faults].
func (s *MethodStub[C]) SetFault(f testkit.Fault[C]) *MethodStub[C] {
	s.fault = f
	return s
}

// FaultsWhen configures a predicate-based fault. The fault fires only when
// pred returns true for the call struct AND the counter fires (every nth
// matching call). Use n=1 to fire on every matching call.
func (s *MethodStub[C]) FaultsWhen(pred func(C) bool, err error, n int) *MethodStub[C] {
	s.fault = testkit.And[C](
		testkit.NewPredicateFault[C](err, pred),
		testkit.NewCountedFault[C](err, n),
	)
	return s
}

// FaultsWithProbability configures probabilistic fault injection. Each call
// has probability p (in [0.0, 1.0]) of faulting. Uses the configured
// [RandSource] (or [DefaultRandSource] if none set).
func (s *MethodStub[C]) FaultsWithProbability(p float64, err error) *MethodStub[C] {
	src := s.rand
	if src == nil {
		src = rand.DefaultRandSource()
	}
	s.fault = testkit.NewProbabilityFault[C](err, p, src)
	return s
}

// FaultsUntil configures a time-windowed fault that fires until the clock
// reaches the given deadline. Uses the configured [Clock] (or real time
// if none set).
func (s *MethodStub[C]) FaultsUntil(deadline time.Time, err error) *MethodStub[C] {
	s.fault = testkit.NewWindowedFault[C](err, deadline)
	return s
}

// FaultsFor configures a time-windowed fault that fires for the given
// duration from now. "Now" is determined by the configured [Clock] (or
// real time if none set).
func (s *MethodStub[C]) FaultsFor(d time.Duration, err error) *MethodStub[C] {
	now := testkit.ClockNow(s.clock)
	s.fault = testkit.NewWindowedFault[C](err, now.Add(d))
	return s
}

// WithClock sets the clock used by time-aware fault strategies, latency
// injection, recorded-call timestamps, and WaitForN/WaitFor timeouts.
// Pass a [TestClock] for deterministic virtual time or a consumer's
// own [Clock] implementation. Default is real time.
//
// The clock is propagated to the embedded [Recorder] so that
// [Recorder.Timestamped] and [Recorder.WaitForN] use the same clock.
func (s *MethodStub[C]) WithClock(clk clock.Clock) *MethodStub[C] {
	s.clock = clk
	s.Recorder.WithClock(clk)
	return s
}

// WithRandSource sets the random number generator used by probabilistic
// fault injection. Pass [FixedRandSource] for deterministic testing or a
// consumer's own seeded [RandSource]. Default is [DefaultRandSource].
func (s *MethodStub[C]) WithRandSource(src rand.Source) *MethodStub[C] {
	s.rand = src
	return s
}

// Latency configures a clock-driven sleep before every dispatch path,
// including fault and unconfigured paths. Use Latency(0) to disable
// (default). Composes with Faults — Latency(5*time.Second).Faults(err, 1)
// models a slow-then-failing backend.
func (s *MethodStub[C]) Latency(d time.Duration) *MethodStub[C] {
	s.latency = d
	return s
}

// SleepLatency sleeps for the configured latency duration. Called by
// generated dispatch code at the start of every method call, before the
// fault check. No-op when no latency is configured.
func (s *MethodStub[C]) SleepLatency() {
	if s.latency <= 0 {
		return
	}
	if s.clock != nil {
		s.clock.Sleep(s.latency)
	} else {
		time.Sleep(s.latency)
	}
}

// Clock returns the configured clock, or nil if using real time.
func (s *MethodStub[C]) Clock() clock.Clock {
	return s.clock
}

// ShouldFaultFor checks whether the fault should fire for the given call.
// Passes the call struct and the configured clock to the underlying [Fault]
// strategy. Generated stubs call this after building the call struct.
func (s *MethodStub[C]) ShouldFaultFor(call C) (bool, error) {
	if s.fault == nil {
		return false, nil
	}
	//nolint:wrapcheck // fault errors must pass through unwrapped for errors.Is
	return s.fault.ShouldFire(call, s.clock)
}

// FailUnexpectedCall fatals the test if strict mode is enabled. Takes the
// typed call struct for structured error messages. Generated stubs call this
// when no behavior (func, returns, fault) is configured for a method call.
func (s *MethodStub[C]) FailUnexpectedCall(call C) {
	if s.strict && s.tb != nil {
		s.tb.Fatalf("%s: unexpected call (strict mode)\n  call: %+v", s.name, call)
	}
}

// Times sets the exact expected number of calls. Checked by [MethodStub.Verify].
func (s *MethodStub[C]) Times(n int) *MethodStub[C] {
	s.times = &n
	return s
}

// TimesAtLeast sets the minimum expected number of calls. Checked by
// [MethodStub.Verify].
func (s *MethodStub[C]) TimesAtLeast(n int) *MethodStub[C] {
	s.atLeast = &n
	return s
}

// Verify checks Times and TimesAtLeast expectations against the actual
// call count. Called automatically via [testing.TB.Cleanup] when tb was
// provided to [NewMethodStub].
func (s *MethodStub[C]) Verify() {
	if s.tb == nil {
		return
	}
	count := s.CallCount()
	if s.times != nil && count != *s.times {
		s.tb.Errorf("%s: expected %d call(s), got %d", s.name, *s.times, count)
	}
	if s.atLeast != nil && count < *s.atLeast {
		s.tb.Errorf("%s: expected at least %d call(s), got %d", s.name, *s.atLeast, count)
	}
}

// Reset clears recorded calls, resets fault counters, and clears
// Times/TimesAtLeast expectations. It does NOT clear Func, Returns,
// or Faults configuration — behavior is preserved, only observations
// are rewound. This matches the pattern: config sticks, counters rewind.
func (s *MethodStub[C]) Reset() {
	s.Recorder.Reset()
	if s.fault != nil {
		s.fault.Reset()
	}
	s.times = nil
	s.atLeast = nil
}

// Name returns the method name (e.g. "Store.Get").
func (s *MethodStub[C]) Name() string {
	return s.name
}

// TB returns the associated testing.TB, or nil.
func (s *MethodStub[C]) TB() testing.TB {
	return s.tb
}
