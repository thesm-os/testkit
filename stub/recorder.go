// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"slices"
	"sync"
	"testing"
	"time"

	"go.thesmos.sh/testkit/clock"
)

// RecordedCall wraps a recorded value with the timestamp of when it was
// recorded. Access via [Recorder.Timestamped].
type RecordedCall[T any] struct {
	Value T
	At    time.Time
}

// Recorder is a thread-safe call log that captures values of type T. It is the
// core observation primitive used by recording wrappers, simulation drivers,
// and integration tests.
//
//	rec := testkit.NewRecorder[PutCall]()
//	rec.Record(PutCall{Key: "a", Value: "1"})
//	calls := rec.Calls() // defensive copy
type Recorder[T any] struct {
	mu      sync.Mutex
	cond    *sync.Cond
	entries []RecordedCall[T]
	hooks   []func(T)
	gate    *Gate
	clock   clock.Clock
	bench   bool
}

// NewRecorder returns a new [Recorder] ready for use.
func NewRecorder[T any]() *Recorder[T] {
	r := &Recorder[T]{}
	r.cond = sync.NewCond(&r.mu)
	return r
}

// WithClock sets the clock used for timestamping recorded calls and for
// timeout computation in [Recorder.WaitForN] and [Recorder.WaitFor].
// Default is real wall-clock time.
func (r *Recorder[T]) WithClock(clk clock.Clock) *Recorder[T] {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clock = clk
	return r
}

// now returns the clock's current time, or time.Now() if no clock is set.
func (r *Recorder[T]) now() time.Time {
	if r.clock != nil {
		return r.clock.Now()
	}
	return time.Now()
}

// BenchMode disables call recording. In bench mode, [Recorder.Record] is a
// no-op — no allocation, no hooks, no gate checks. Dispatch (Func, Returns,
// Faults) still works normally through the enclosing [MethodStub].
func (r *Recorder[T]) BenchMode() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bench = true
}

// IsBenchMode reports whether [BenchMode] has been enabled. Generated
// dispatch reads this to skip auxiliary work that would otherwise show
// up in benchmark allocation counts — notably the //testkit:deprecated
// log line, which calls [testing.TB.Logf] and allocates the formatted
// message every dispatch.
//
// Reads `r.bench` without the mutex, matching [Recorder.Record]'s
// fast-path. Callers must set BenchMode during single-threaded
// construction (i.e. before invoking the stub from goroutines), the
// same contract Record relies on.
func (r *Recorder[T]) IsBenchMode() bool {
	return r.bench
}

// Record appends v to the call log. If a [Gate] is active, Record blocks
// until the gate releases. Hooks are fired synchronously after recording
// (under the mutex — keep them fast). In bench mode, Record is a no-op.
func (r *Recorder[T]) Record(v T) {
	if r.bench {
		return
	}
	if g := r.activeGate(); g != nil {
		g.wait()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, RecordedCall[T]{Value: v, At: r.now()})
	for _, h := range r.hooks {
		h(v)
	}
	r.cond.Broadcast()
}

// Calls returns a defensive copy of all recorded values.
func (r *Recorder[T]) Calls() []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]T, len(r.entries))
	for i, e := range r.entries {
		cp[i] = e.Value
	}
	return cp
}

// Timestamped returns a defensive copy of all recorded entries with timestamps.
func (r *Recorder[T]) Timestamped() []RecordedCall[T] {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]RecordedCall[T], len(r.entries))
	copy(cp, r.entries)
	return cp
}

// CallCount returns the number of recorded values.
func (r *Recorder[T]) CallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entries)
}

// LastCall returns the most recently recorded value. It calls tb.Fatalf if
// no calls have been recorded.
func (r *Recorder[T]) LastCall(tb testing.TB) T {
	tb.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.entries) == 0 {
		tb.Fatalf("LastCall: no calls recorded")
		var zero T
		return zero
	}
	return r.entries[len(r.entries)-1].Value
}

// Reset clears all recorded calls. Hooks, gates, and clock are preserved.
func (r *Recorder[T]) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = nil
}

// --- Assertion helpers ---

// AssertCalledOnce calls tb.Fatalf unless exactly one call has been recorded.
// Returns the single recorded value.
func (r *Recorder[T]) AssertCalledOnce(tb testing.TB, msg string) T {
	tb.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.entries) != 1 {
		tb.Fatalf("%s: expected 1 call, got %d", msg, len(r.entries))
		var zero T
		return zero
	}
	return r.entries[0].Value
}

// AssertCalledN calls tb.Fatalf unless exactly n calls have been recorded.
// Returns all recorded values.
func (r *Recorder[T]) AssertCalledN(tb testing.TB, n int, msg string) []T {
	tb.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.entries) != n {
		tb.Fatalf("%s: expected %d calls, got %d", msg, n, len(r.entries))
		return nil
	}
	cp := make([]T, len(r.entries))
	for i, e := range r.entries {
		cp[i] = e.Value
	}
	return cp
}

// AssertNotCalled calls tb.Fatalf if any calls have been recorded.
func (r *Recorder[T]) AssertNotCalled(tb testing.TB, msg string) {
	tb.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.entries) != 0 {
		tb.Fatalf("%s: expected no calls, got %d", msg, len(r.entries))
	}
}

// --- Filtering ---

// Filter returns all recorded values that satisfy pred.
func (r *Recorder[T]) Filter(pred func(T) bool) []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []T
	for _, e := range r.entries {
		if pred(e.Value) {
			out = append(out, e.Value)
		}
	}
	return out
}

// First returns the first recorded value that satisfies pred, or false if
// none match.
func (r *Recorder[T]) First(pred func(T) bool) (T, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		if pred(e.Value) {
			return e.Value, true
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
	for _, e := range r.entries {
		if !pred(e.Value) {
			return false
		}
	}
	return true
}

// --- Waiting ---

// WaitForN blocks until at least n calls have been recorded, or until timeout
// expires. Calls tb.Fatalf on timeout. Uses condition variables internally —
// no polling. Routes timeouts through the configured [Clock].
func (r *Recorder[T]) WaitForN(tb testing.TB, n int, timeout time.Duration) {
	tb.Helper()
	now := r.now()
	deadline := now.Add(timeout)
	r.mu.Lock()
	defer r.mu.Unlock()
	for len(r.entries) < n {
		remaining := deadline.Sub(r.now())
		if remaining <= 0 {
			tb.Fatalf("WaitForN: timed out waiting for %d calls (got %d)", n, len(r.entries))
			return
		}
		// Use a timer to wake up the cond.Wait if the deadline passes.
		timer := r.afterFunc(remaining, func() {
			r.cond.Broadcast()
		})
		r.cond.Wait()
		timer.Stop()
	}
}

// WaitFor blocks until a recorded value satisfies pred, or until timeout
// expires. Calls tb.Fatalf on timeout. Routes timeouts through the
// configured [Clock].
func (r *Recorder[T]) WaitFor(tb testing.TB, pred func(T) bool, timeout time.Duration, msg string) {
	tb.Helper()
	now := r.now()
	deadline := now.Add(timeout)
	r.mu.Lock()
	defer r.mu.Unlock()
	for {
		if slices.ContainsFunc(r.entries, func(e RecordedCall[T]) bool { return pred(e.Value) }) {
			return
		}
		remaining := deadline.Sub(r.now())
		if remaining <= 0 {
			tb.Fatalf("%s: timed out waiting for matching call", msg)
			return
		}
		timer := r.afterFunc(remaining, func() {
			r.cond.Broadcast()
		})
		r.cond.Wait()
		timer.Stop()
	}
}

// afterFunc creates a timer using the configured clock. Falls back to
// time.AfterFunc when no clock is set. The returned timer's Stop method
// cleans up the wrapper goroutine to prevent leaks.
func (r *Recorder[T]) afterFunc(d time.Duration, f func()) stoppable {
	if r.clock != nil {
		t := r.clock.NewTimer(d)
		done := make(chan struct{})
		go func() {
			select {
			case <-t.C():
				f()
			case <-done:
			}
		}()
		return &cancelableTimer{inner: t, done: done}
	}
	return &realAfterFunc{inner: time.AfterFunc(d, f)}
}

// stoppable is the subset of Timer used by afterFunc callers.
type stoppable interface {
	Stop() bool
}

// cancelableTimer wraps a virtual-clock Timer with a done channel that
// cancels the wrapper goroutine on Stop, preventing leaks.
type cancelableTimer struct {
	inner clock.Timer
	done  chan struct{}
	once  sync.Once
}

func (t *cancelableTimer) Stop() bool {
	t.once.Do(func() { close(t.done) })
	return t.inner.Stop()
}

// realAfterFunc wraps time.AfterFunc to satisfy stoppable.
type realAfterFunc struct {
	inner *time.Timer
}

func (t *realAfterFunc) Stop() bool {
	return t.inner.Stop()
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
