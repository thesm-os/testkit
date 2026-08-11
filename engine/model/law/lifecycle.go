// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
)

// IdempotentLifecycle verifies that calling the lifecycle method
// twice in succession is observably equivalent to calling it once.
// Auto-emitted for Lifecycle methods carrying //testkit:idempotent.
type IdempotentLifecycle[T any, Obs any] struct {
	Call    func(*rapid.T, T) error
	Observe func(*rapid.T, T) Obs
}

// ID returns the stable identifier for this law.
func (IdempotentLifecycle[T, Obs]) ID() string { return lawid.IdempotentLifecycle }

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
	Open  func(*rapid.T, T) error
	Close func(*rapid.T, T) error
	// Cycles is how many open/close rounds to run. Zero defaults to 16 — a
	// leak of one goroutine per cycle needs repetition to rise above the
	// scheduler's own noise.
	Cycles int
	// Tolerance is the goroutine growth treated as noise rather than a leak.
	// Zero defaults to 4, which absorbs the runtime's own background workers
	// without absorbing a per-cycle leak.
	Tolerance int
}

// ID returns the stable identifier for this law.
func (LeakFree[T]) ID() string { return lawid.LeakFree }

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

// LifecycleRespectsContext verifies that a Lifecycle method invoked
// with an already-cancelled context returns a context error instead
// of proceeding. Auto-emitted for Lifecycle methods taking a context.
//
// The law calls Op with a pre-cancelled context and requires the
// result to satisfy errors.Is(err, context.Canceled). An
// implementation that ignores the context and returns success fails.
type LifecycleRespectsContext[T any] struct {
	Op func(ctx context.Context, sut T) error
}

// ID returns the stable identifier for this law.
func (LifecycleRespectsContext[T]) ID() string { return lawid.LifecycleRespectsContext }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (LifecycleRespectsContext[T]) REQID() string { return "" }

// Check invokes Op with a cancelled context and verifies it reports
// the cancellation.
func (l LifecycleRespectsContext[T]) Check(rt *rapid.T, sut, _ T) error {
	ctx, cancel := context.WithCancel(rt.Context())
	cancel()
	err := l.Op(ctx, sut)
	if !errors.Is(err, context.Canceled) {
		return fmt.Errorf(
			"LifecycleRespectsContext: op with cancelled context returned %v (want context.Canceled)",
			err,
		)
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
func (PoisonNilOnFresh[T]) ID() string { return lawid.PoisonNilOnFresh }

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
func (PoisonIdempotentRead[T]) ID() string { return lawid.PoisonIdempotentRead }

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

// PoisonConsistent verifies that a poison condition is sticky: once
// the accessor reports poison, it keeps reporting it across
// subsequent reads rather than spontaneously healing. Auto-emitted
// for PoisonAccessor methods.
//
// Poison induces the condition; Probe reads it. The law establishes
// poison, confirms it took (a nil probe means the induction was a
// no-op and the law holds vacuously), then probes Reads more times
// (default 3) and fails if any returns nil. Distinct from
// [PoisonIdempotentRead], which checks read purity rather than
// stickiness after a poisoning event.
type PoisonConsistent[T any] struct {
	Poison func(*rapid.T, T)
	Probe  func(*rapid.T, T) error
	// Reads is how many times to probe the poisoned state. Zero defaults to
	// 3: two agreeing reads can agree by accident, three rarely do.
	Reads int
}

// ID returns the stable identifier for this law.
func (PoisonConsistent[T]) ID() string { return lawid.PoisonConsistent }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PoisonConsistent[T]) REQID() string { return "" }

// Check induces poison and verifies it persists across follow-up
// probes.
func (l PoisonConsistent[T]) Check(rt *rapid.T, sut, _ T) error {
	l.Poison(rt, sut)
	if l.Probe(rt, sut) == nil {
		return nil // poisoning was a no-op; nothing to hold consistent
	}
	n := l.Reads
	if n <= 0 {
		n = 3
	}
	for i := range n {
		if err := l.Probe(rt, sut); err == nil {
			return fmt.Errorf("PoisonConsistent: poison healed spontaneously (probe %d after poison returned nil)", i+1)
		}
	}
	return nil
}

// LifecycleAfterCloseSentinel verifies that once the lifecycle's
// Close has run, the paired method rejects further use with the
// configured sentinel error. Auto-emitted for the
// //testkit:lifecycle-after-close <Reader> directive. (The cursor
// composite has its own narrower variant,
// [CursorNextAfterCloseSentinel].)
type LifecycleAfterCloseSentinel[T any] struct {
	Close    func(*rapid.T, T) error
	Op       func(*rapid.T, T) error
	Sentinel error
}

// ID returns the stable identifier for this law.
func (LifecycleAfterCloseSentinel[T]) ID() string { return lawid.LifecycleAfterClose }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (LifecycleAfterCloseSentinel[T]) REQID() string { return "" }

// Check closes the SUT and verifies the paired method returns the
// sentinel afterwards.
func (l LifecycleAfterCloseSentinel[T]) Check(rt *rapid.T, sut, _ T) error {
	if err := l.Close(rt, sut); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	err := l.Op(rt, sut)
	if !errors.Is(err, l.Sentinel) {
		return fmt.Errorf("LifecycleAfterCloseSentinel: op after close returned %v (want %v)", err, l.Sentinel)
	}
	return nil
}
