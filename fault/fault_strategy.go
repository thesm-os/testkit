// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fault

import (
	"sync"
	"time"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/rand"
)

// Fault evaluates whether a fault should fire for a given call. Implementations
// are composable strategies — counter-based, probabilistic, time-windowed, or
// predicate-based. [MethodStub] holds a single Fault and delegates to it on
// every dispatch.
//
// The call parameter is the about-to-be-recorded call struct (params only,
// results not yet populated). The clock parameter is the stub's configured
// clock (nil means real time). Implementations that don't need the call or
// clock simply ignore them.
type Fault[C any] interface {
	// ShouldFire reports whether the fault should fire for this call.
	// Returns (true, faultErr) to fire, (false, nil) to pass through.
	ShouldFire(call C, clock clock.Clock) (bool, error)

	// Reset rewinds internal counters without clearing configuration.
	// Matches the MethodStub.Reset contract: config sticks, counters rewind.
	Reset()
}

// stateless supplies the no-op Reset shared by strategies that hold no counter
// state.
//
// [Fault.Reset]'s contract is "rewind counters, keep configuration". A strategy
// whose decision comes entirely from its configuration — a probability, a
// deadline, a predicate — has nothing to rewind, so Reset exists only to
// satisfy the interface. Embedding one implementation states that intent once
// instead of repeating an empty body per strategy.
type stateless struct{}

// Reset is a no-op: the embedding strategy holds no counter to rewind.
func (stateless) Reset() {}

// CountedFault fires on every Nth call, regardless of the call value or clock.
// This is the strategy behind [MethodStub.Faults].
type CountedFault[C any] struct {
	inner faultInjector
}

// NewCountedFault returns a [CountedFault] that fires on every nth call.
// A value of n <= 0 disables the fault (ShouldFire always returns false).
func NewCountedFault[C any](err error, n int) *CountedFault[C] {
	return &CountedFault[C]{
		inner: newFaultInjector(err, n),
	}
}

// ShouldFire increments the internal counter and reports whether the fault
// should fire. The call and clock parameters are ignored — counted faults
// are context-free.
func (f *CountedFault[C]) ShouldFire(_ C, _ clock.Clock) (bool, error) {
	if f.inner.shouldFire() {
		return true, f.inner.faultErr
	}
	return false, nil
}

// Reset zeroes the call counter without clearing the error or interval.
func (f *CountedFault[C]) Reset() {
	f.inner.reset()
}

// RetryFault fires for the first N-1 calls and succeeds on the Nth call.
// This models the canonical resilience-loop pattern: "first N-1 attempts
// fail, Nth attempt succeeds." This is the strategy behind the generated
// RetrySchedule helper from the retry-succeeds-on-attempt directive.
//
// Different from [CountedFault] which fires periodically (every Nth call).
// RetryFault is finite — once the Nth call succeeds, all subsequent calls
// succeed.
type RetryFault[C any] struct {
	err   error
	n     int // succeed on this call number
	mu    sync.Mutex
	count int
}

// NewRetryFault returns a [RetryFault] that fails for the first n-1 calls
// and succeeds on the nth call and all subsequent calls.
func NewRetryFault[C any](err error, n int) *RetryFault[C] {
	return &RetryFault[C]{err: err, n: n}
}

// ShouldFire fails for calls 1 through n-1, succeeds from call n onward.
func (f *RetryFault[C]) ShouldFire(_ C, _ clock.Clock) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count++
	if f.count < f.n {
		return true, f.err
	}
	return false, nil
}

// Reset rewinds the call counter.
func (f *RetryFault[C]) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count = 0
}

// ProbabilityFault fires with probability p on each call. Uses the configured
// [RandSource] to generate random numbers. This is the strategy behind
// [MethodStub.FaultsWithProbability].
type ProbabilityFault[C any] struct {
	stateless
	err error
	p   float64
	src rand.Source
}

// NewProbabilityFault returns a [ProbabilityFault] that fires with probability p
// (in [0.0, 1.0]) on each call. The [RandSource] determines the random number
// generation — pass [DefaultRandSource] for stdlib or [FixedRandSource] for
// deterministic testing.
func NewProbabilityFault[C any](err error, p float64, src rand.Source) *ProbabilityFault[C] {
	return &ProbabilityFault[C]{err: err, p: p, src: src}
}

// ShouldFire draws a random number and fires if it falls below p.
func (f *ProbabilityFault[C]) ShouldFire(_ C, _ clock.Clock) (bool, error) {
	if f.src.Float64() < f.p {
		return true, f.err
	}
	return false, nil
}

// WindowedFault fires until the clock advances past a deadline. This is the
// strategy behind [MethodStub.FaultsUntil] and [MethodStub.FaultsFor].
// The deadline is set at construction and never modified — no synchronization
// needed.
type WindowedFault[C any] struct {
	stateless
	err      error
	deadline time.Time
}

// NewWindowedFault returns a [WindowedFault] that fires until the given
// deadline. When clock.Now() >= deadline, the fault stops firing.
func NewWindowedFault[C any](err error, deadline time.Time) *WindowedFault[C] {
	return &WindowedFault[C]{err: err, deadline: deadline}
}

// ShouldFire fires if the clock's current time is before the deadline.
// If clock is nil, uses real time.
func (f *WindowedFault[C]) ShouldFire(_ C, clock clock.Clock) (bool, error) {
	now := ClockNow(clock)
	if now.Before(f.deadline) {
		return true, f.err
	}
	return false, nil
}

// PredicateFault fires when a predicate returns true for the call value.
// This is the strategy behind [MethodStub.FaultsWhen].
type PredicateFault[C any] struct {
	stateless
	err  error
	pred func(C) bool
}

// NewPredicateFault returns a [PredicateFault] that fires when pred returns
// true for the about-to-be-recorded call struct.
func NewPredicateFault[C any](err error, pred func(C) bool) *PredicateFault[C] {
	return &PredicateFault[C]{err: err, pred: pred}
}

// ShouldFire evaluates the predicate against the call value.
func (f *PredicateFault[C]) ShouldFire(call C, _ clock.Clock) (bool, error) {
	if f.pred(call) {
		return true, f.err
	}
	return false, nil
}

// AndFault composes multiple [Fault] strategies with AND semantics — the
// fault fires only when ALL inner strategies fire. The error from the first
// strategy that fires is returned.
//
// Inner faults are evaluated left-to-right with short-circuit: if an inner
// fault does not fire, later faults are not evaluated. This matters when
// inner faults have side effects (e.g., [CountedFault] advances its counter
// on every ShouldFire call). In And(predicate, counted), the counter
// advances only on calls that pass the predicate — so FaultsWhen(pred, err, 3)
// fires on the 3rd *matching* call, not the 3rd call overall.
//
// Use this for patterns like "fail only when this call is for run-1 AND
// we're within the first 5 seconds":
//
//	testkit.And(predFault, windowFault)
type AndFault[C any] struct {
	inner []Fault[C]
}

// And composes faults with AND semantics. All must fire for the composed
// fault to fire.
func And[C any](faults ...Fault[C]) *AndFault[C] {
	return &AndFault[C]{inner: faults}
}

// ShouldFire returns true only when every inner fault fires.
func (f *AndFault[C]) ShouldFire(call C, clock clock.Clock) (bool, error) {
	var firstErr error
	for _, inner := range f.inner {
		fired, err := inner.ShouldFire(call, clock)
		if !fired {
			return false, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return true, firstErr //nolint:wrapcheck // fault errors must pass through unwrapped for errors.Is
}

// Reset resets all inner faults.
func (f *AndFault[C]) Reset() {
	for _, inner := range f.inner {
		inner.Reset()
	}
}

// OrFault composes multiple [Fault] strategies with OR semantics — the
// fault fires when ANY inner strategy fires. The error from the first
// strategy that fires is returned.
type OrFault[C any] struct {
	inner []Fault[C]
}

// Or composes faults with OR semantics. Any firing triggers the composed fault.
func Or[C any](faults ...Fault[C]) *OrFault[C] {
	return &OrFault[C]{inner: faults}
}

// ShouldFire returns true when any inner fault fires.
func (f *OrFault[C]) ShouldFire(call C, clock clock.Clock) (bool, error) {
	for _, inner := range f.inner {
		fired, err := inner.ShouldFire(call, clock)
		if fired {
			//nolint:wrapcheck // fault errors pass through unwrapped for errors.Is
			return true, err
		}
	}
	return false, nil
}

// Reset resets all inner faults.
func (f *OrFault[C]) Reset() {
	for _, inner := range f.inner {
		inner.Reset()
	}
}

// ClockNow returns clock.Now() if clock is non-nil, otherwise time.Now().
func ClockNow(clock clock.Clock) time.Time {
	if clock != nil {
		return clock.Now()
	}
	return time.Now()
}
