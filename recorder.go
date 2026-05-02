// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"slices"
	"sync"
	"testing"
	"time"
)

// Recorder is a thread-safe call log that captures values of type T. It is the
// core observation primitive used by recording wrappers, simulation drivers,
// and integration tests.
//
//	rec := testkit.NewRecorder[PutCall]()
//	rec.Record(PutCall{Key: "a", Value: "1"})
//	calls := rec.Calls() // defensive copy
type Recorder[T any] struct {
	mu    sync.Mutex
	cond  *sync.Cond
	calls []T
	hooks []func(T)
	gate  *Gate
}

// NewRecorder returns a new [Recorder] ready for use.
func NewRecorder[T any]() *Recorder[T] {
	r := &Recorder[T]{}
	r.cond = sync.NewCond(&r.mu)
	return r
}

// Record appends v to the call log. If a [Gate] is active, Record blocks
// until the gate releases. Hooks are fired synchronously after recording
// (under the mutex — keep them fast).
func (r *Recorder[T]) Record(v T) {
	if g := r.activeGate(); g != nil {
		g.wait()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, v)
	for _, h := range r.hooks {
		h(v)
	}
	r.cond.Broadcast()
}

// Calls returns a defensive copy of all recorded values.
func (r *Recorder[T]) Calls() []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]T, len(r.calls))
	copy(cp, r.calls)
	return cp
}

// CallCount returns the number of recorded values.
func (r *Recorder[T]) CallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

// LastCall returns the most recently recorded value. It calls tb.Fatalf if
// no calls have been recorded.
func (r *Recorder[T]) LastCall(tb testing.TB) T {
	tb.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		tb.Fatalf("LastCall: no calls recorded")
		var zero T
		return zero
	}
	return r.calls[len(r.calls)-1]
}

// Reset clears all recorded calls. Hooks and gates are preserved.
func (r *Recorder[T]) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = nil
}

// --- Assertion helpers ---

// AssertCalledOnce calls tb.Fatalf unless exactly one call has been recorded.
// Returns the single recorded value.
func (r *Recorder[T]) AssertCalledOnce(tb testing.TB, msg string) T {
	tb.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) != 1 {
		tb.Fatalf("%s: expected 1 call, got %d", msg, len(r.calls))
		var zero T
		return zero
	}
	return r.calls[0]
}

// AssertCalledN calls tb.Fatalf unless exactly n calls have been recorded.
// Returns all recorded values.
func (r *Recorder[T]) AssertCalledN(tb testing.TB, n int, msg string) []T {
	tb.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) != n {
		tb.Fatalf("%s: expected %d calls, got %d", msg, n, len(r.calls))
		return nil
	}
	cp := make([]T, len(r.calls))
	copy(cp, r.calls)
	return cp
}

// AssertNotCalled calls tb.Fatalf if any calls have been recorded.
func (r *Recorder[T]) AssertNotCalled(tb testing.TB, msg string) {
	tb.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) != 0 {
		tb.Fatalf("%s: expected no calls, got %d", msg, len(r.calls))
	}
}

// --- Filtering ---

// Filter returns all recorded values that satisfy pred.
func (r *Recorder[T]) Filter(pred func(T) bool) []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []T
	for _, c := range r.calls {
		if pred(c) {
			out = append(out, c)
		}
	}
	return out
}

// First returns the first recorded value that satisfies pred, or false if
// none match.
func (r *Recorder[T]) First(pred func(T) bool) (T, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.calls {
		if pred(c) {
			return c, true
		}
	}
	var zero T
	return zero, false
}

// Any reports whether any recorded value satisfies pred.
func (r *Recorder[T]) Any(pred func(T) bool) bool {
	_, found := r.First(pred)
	return found
}

// All reports whether every recorded value satisfies pred. Returns true for
// an empty call log.
func (r *Recorder[T]) All(pred func(T) bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.calls {
		if !pred(c) {
			return false
		}
	}
	return true
}

// --- Waiting ---

// WaitForN blocks until at least n calls have been recorded, or until timeout
// expires. Calls tb.Fatalf on timeout. Uses condition variables internally —
// no polling.
func (r *Recorder[T]) WaitForN(tb testing.TB, n int, timeout time.Duration) {
	tb.Helper()
	deadline := time.Now().Add(timeout)
	r.mu.Lock()
	defer r.mu.Unlock()
	for len(r.calls) < n {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			tb.Fatalf("WaitForN: timed out waiting for %d calls (got %d)", n, len(r.calls))
			return
		}
		// Use a timer to wake up the cond.Wait if the deadline passes.
		done := make(chan struct{})
		timer := time.AfterFunc(remaining, func() {
			r.cond.Broadcast()
			close(done)
		})
		r.cond.Wait()
		timer.Stop()
	}
}

// WaitFor blocks until a recorded value satisfies pred, or until timeout
// expires. Calls tb.Fatalf on timeout.
func (r *Recorder[T]) WaitFor(tb testing.TB, pred func(T) bool, timeout time.Duration, msg string) {
	tb.Helper()
	deadline := time.Now().Add(timeout)
	r.mu.Lock()
	defer r.mu.Unlock()
	for {
		if slices.ContainsFunc(r.calls, pred) {
			return
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			tb.Fatalf("%s: timed out waiting for matching call", msg)
			return
		}
		done := make(chan struct{})
		timer := time.AfterFunc(remaining, func() {
			r.cond.Broadcast()
			close(done)
		})
		r.cond.Wait()
		timer.Stop()
	}
}

// --- Hooks ---

// OnRecord registers a callback that fires synchronously on every
// [Recorder.Record] call, under the mutex. Keep hooks fast — offload
// expensive work to a channel.
func (r *Recorder[T]) OnRecord(fn func(T)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, fn)
}

// --- Gating ---

// Gate blocks [Recorder.Record] calls until released. Use gates to create
// deterministic race conditions in tests.
type Gate struct {
	ch chan struct{}
}

// NewGate creates and activates a [Gate] on this recorder. While a gate is
// active, [Recorder.Record] blocks until the gate is released. Only one gate
// may be active at a time.
func (r *Recorder[T]) NewGate() *Gate {
	r.mu.Lock()
	defer r.mu.Unlock()
	g := &Gate{ch: make(chan struct{})}
	r.gate = g
	return g
}

// Release unblocks all goroutines waiting on this gate.
func (g *Gate) Release() {
	close(g.ch)
}

// ReleaseOne unblocks exactly one goroutine waiting on this gate by sending
// a single value.
func (g *Gate) ReleaseOne() {
	g.ch <- struct{}{}
}

func (r *Recorder[T]) activeGate() *Gate {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.gate
}

func (g *Gate) wait() {
	<-g.ch
}
