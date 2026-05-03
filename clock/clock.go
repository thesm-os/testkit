// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package clock

import "time"

// Clock abstracts time operations for deterministic testing. The default is
// real wall-clock time ([RealClock]). Consumers inject a [TestClock] or their
// own virtual clock (e.g., a simulation engine's clock) via [MethodStub.WithClock]
// or the generated stub's WithClock constructor option.
//
// testkit defines the interface; consumers own the implementation. Standard
// dependency inversion — testkit never imports consumer packages.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// Sleep blocks for at least duration d. With a virtual clock,
	// Sleep advances virtual time without real-time blocking.
	Sleep(d time.Duration)

	// After waits for duration d and then sends the current time on the
	// returned channel.
	After(d time.Duration) <-chan time.Time

	// NewTimer creates a new Timer that fires after duration d.
	NewTimer(d time.Duration) Timer
}

// Timer abstracts [time.Timer] for use with [Clock]. Real timers wrap
// [time.Timer]; virtual timers are driven by [TestClock.Advance].
type Timer interface {
	// C returns the channel on which the timer fires.
	C() <-chan time.Time

	// Stop prevents the timer from firing. Returns true if the call
	// stops the timer, false if it has already fired or been stopped.
	Stop() bool

	// Reset changes the timer to fire after duration d. Returns true
	// if the timer had been active, false if it had fired or been stopped.
	Reset(d time.Duration) bool
}

// realClock implements [Clock] using the standard library time package.
type realClock struct{}

// RealClock returns a [Clock] backed by the standard library time package.
// This is the default clock used when no clock is explicitly configured.
func RealClock() Clock {
	return realClock{}
}

func (realClock) Now() time.Time                         { return time.Now() }
func (realClock) Sleep(d time.Duration)                  { time.Sleep(d) }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (realClock) NewTimer(d time.Duration) Timer         { return &realTimer{inner: time.NewTimer(d)} }

// realTimer wraps [time.Timer] to satisfy [Timer].
type realTimer struct {
	inner *time.Timer
}

func (t *realTimer) C() <-chan time.Time        { return t.inner.C }
func (t *realTimer) Stop() bool                 { return t.inner.Stop() }
func (t *realTimer) Reset(d time.Duration) bool { return t.inner.Reset(d) }
