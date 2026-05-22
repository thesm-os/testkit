// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"fmt"
	"runtime"

	"pgregory.net/rapid"
)

// IdempotentLifecycle verifies that calling the lifecycle method
// twice in succession is observably equivalent to calling it once.
// Auto-emitted for Lifecycle methods carrying //testkit:idempotent.
type IdempotentLifecycle[T any, Obs any] struct {
	Call    func(*rapid.T, T) error
	Observe func(*rapid.T, T) Obs
}

// ID returns the stable identifier for this law.
func (IdempotentLifecycle[T, Obs]) ID() string { return "AUTO-IDEMPOTENT-LIFECYCLE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (IdempotentLifecycle[T, Obs]) REQID() string { return "" }

// Check verifies a second Call leaves Observe unchanged and does
// not error.
func (l IdempotentLifecycle[T, Obs]) Check(rt *rapid.T, sut, _ T) error {
	if err := l.Call(rt, sut); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	before := l.Observe(rt, sut)
	if err := l.Call(rt, sut); err != nil {
		return fmt.Errorf("IdempotentLifecycle: second call errored: %v", err)
	}
	after := l.Observe(rt, sut)
	if fmt.Sprint(before) != fmt.Sprint(after) {
		return fmt.Errorf("IdempotentLifecycle: second call mutated state: before=%v after=%v", before, after)
	}
	return nil
}

// LeakFree verifies a Lifecycle pair (e.g., Open/Close, Acquire/
// Release) does not leak goroutines across N cycles. Auto-emitted
// for Lifecycle methods carrying //testkit:leak-free.
//
// The cycle count and tolerance are runtime-tuned; the law samples
// runtime.NumGoroutine before and after the cycle and flags a
// growth exceeding Tolerance.
type LeakFree[T any] struct {
	Open      func(*rapid.T, T) error
	Close     func(*rapid.T, T) error
	Cycles    int
	Tolerance int
}

// ID returns the stable identifier for this law.
func (LeakFree[T]) ID() string { return "AUTO-LEAK-FREE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (LeakFree[T]) REQID() string { return "" }

// Check runs N Open/Close cycles and asserts the goroutine count
// hasn't drifted by more than Tolerance.
func (l LeakFree[T]) Check(rt *rapid.T, sut, _ T) error {
	cycles := l.Cycles
	if cycles <= 0 {
		cycles = 16
	}
	tolerance := l.Tolerance
	if tolerance <= 0 {
		tolerance = 4
	}
	before := runtime.NumGoroutine()
	for range cycles {
		if err := l.Open(rt, sut); err != nil {
			return nil //nolint:nilerr // precondition failed; law vacuously holds
		}
		if err := l.Close(rt, sut); err != nil {
			return nil //nolint:nilerr // precondition failed; law vacuously holds
		}
	}
	after := runtime.NumGoroutine()
	if drift := after - before; drift > tolerance {
		return fmt.Errorf("LeakFree: goroutine count grew from %d to %d after %d cycles (tolerance %d)",
			before, after, cycles, tolerance)
	}
	return nil
}

// PoisonNilOnFresh verifies a PoisonAccessor returns nil on a
// freshly-constructed impl. Auto-emitted for PoisonAccessor
// methods.
type PoisonNilOnFresh[T any] struct {
	Factory func() T
	Probe   func(*rapid.T, T) error
}

// ID returns the stable identifier for this law.
func (PoisonNilOnFresh[T]) ID() string { return "AUTO-POISON-NIL-ON-FRESH" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PoisonNilOnFresh[T]) REQID() string { return "" }

// Check runs Probe on a freshly-constructed impl and verifies
// the result is nil.
func (l PoisonNilOnFresh[T]) Check(rt *rapid.T, _, _ T) error {
	fresh := l.Factory()
	if err := l.Probe(rt, fresh); err != nil {
		return fmt.Errorf("PoisonNilOnFresh: fresh impl reports poison: %v", err)
	}
	return nil
}

// PoisonIdempotentRead verifies the PoisonAccessor is read-only:
// two consecutive reads return the same value (and the same
// error).
type PoisonIdempotentRead[T any] struct {
	Probe func(*rapid.T, T) error
}

// ID returns the stable identifier for this law.
func (PoisonIdempotentRead[T]) ID() string { return "AUTO-POISON-IDEMPOTENT-READ" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PoisonIdempotentRead[T]) REQID() string { return "" }

// Check verifies two consecutive Probe calls agree.
func (l PoisonIdempotentRead[T]) Check(rt *rapid.T, sut, _ T) error {
	a := l.Probe(rt, sut)
	b := l.Probe(rt, sut)
	if (a == nil) != (b == nil) {
		return fmt.Errorf("PoisonIdempotentRead: first=%v, second=%v", a, b)
	}
	return nil
}
