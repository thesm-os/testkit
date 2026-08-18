// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"context"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"testing"
)

// FailableTB captures a failure instead of aborting the test process, so an
// assertion helper can be driven against input it is expected to reject and the
// rejection observed.
//
//	f := testkit.NewFailableTB()
//	testkit.Equal(f, 1, 2, "must match")
//	if !f.Failed() { t.Fatal("Equal should have failed") }
//
// It stands in for [testing.TB] and for [BenchTB] both. The second is what lets
// a benchmark contract be checked the same way an assertion is: [Loop] returns
// true a bounded number of times, [ReportMetric] records rather than prints, and
// a violated ceiling lands in [Msg] instead of failing the run.
//
//	f := testkit.NewFailableTB().WithIterations(64)
//	c := testkit.StartContract(f).AllocsMax(0)
//	for c.Loop() { sink = make([]byte, 64) }
//	c.End()
//	if !f.Failed() { t.Fatal("an allocating loop must violate AllocsMax(0)") }
//
// One caveat on that direction. Proving a ceiling *rejects* a violating body is
// reliable; proving it *accepts* a compliant one is not, because
// [runtime.MemStats] counts allocations process-wide and a parallel test
// contributes to them.
type FailableTB struct {
	testing.TB // embedded nil interface — panics on unimplemented methods

	mu             sync.Mutex
	ctx            context.Context //nolint:containedctx // test double must hold ctx to implement testing.TB.Context
	cancel         context.CancelFunc
	name           string
	msg            string
	logs           []string
	cleanups       []func()
	metrics        map[string]float64
	helperCalls    int
	iterations     int // how many times Loop is to return true
	looped         int // how many times it has
	failed         bool
	goexit         bool // when true, Fatal/FailNow call runtime.Goexit()
	allocsReported bool

	// spawned tracks the goroutines Go launched, so RunCleanups can
	// join them before the cleanup phase reads the verdict.
	spawned sync.WaitGroup
}

// NewFailableTB returns a new [FailableTB] ready for use. The returned
// instance carries a cancellable [context.Context] accessible via
// [FailableTB.Context], matching the Go 1.24+ [testing.T.Context] semantics.
func NewFailableTB() *FailableTB {
	ctx, cancel := context.WithCancel(context.Background())
	return &FailableTB{
		name:   "FailableTB",
		ctx:    ctx,
		cancel: cancel,
	}
}

// WithGoexit configures Fatal, Fatalf, and FailNow to call
// [runtime.Goexit] after recording the failure. This matches
// [*testing.T] semantics and is required for libraries like
// [pgregory.net/rapid] that expect Fatal to terminate the goroutine.
// Must be called in a dedicated goroutine — Goexit terminates
// only the calling goroutine.
func (f *FailableTB) WithGoexit() *FailableTB {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.goexit = true
	return f
}

// WithIterations bounds how many times [FailableTB.Loop] returns true, which is
// what makes a benchmark body run a known number of times under a stand-in.
//
// The default is zero, so a contract driven without this runs no iterations —
// which is the right default for checking that [Contract.End] rejects being
// called before the loop, and the wrong one for everything else. Sixty-four is
// enough for an allocation delta to clear measurement noise.
func (f *FailableTB) WithIterations(n int) *FailableTB {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.iterations = n
	return f
}

// WithName sets the test name returned by [FailableTB.Name].
func (f *FailableTB) WithName(name string) *FailableTB {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.name = name
	return f
}

// Failed reports whether [FailableTB.Fatal], [FailableTB.Fatalf], or
// [FailableTB.Errorf] was called.
func (f *FailableTB) Failed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.failed
}

// Msg returns the formatted message from the first fatal call, or the most
// recent [FailableTB.Errorf] call.
func (f *FailableTB) Msg() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.msg
}

// Logs returns a defensive copy of all messages passed to [FailableTB.Logf].
func (f *FailableTB) Logs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]string, len(f.logs))
	copy(cp, f.logs)
	return cp
}

// Name returns the test name set by [FailableTB.WithName], defaulting to
// "FailableTB".
func (f *FailableTB) Name() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.name
}

// Helper increments the helper call counter. It does not affect stack traces.
func (f *FailableTB) Helper() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.helperCalls++
}

// HelperCalls returns the number of times [FailableTB.Helper] was called.
func (f *FailableTB) HelperCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.helperCalls
}

// Fatalf records a failure and cancels the context. When [WithGoexit]
// is set, also terminates the goroutine via [runtime.Goexit].
// Subsequent calls are ignored — the first failure wins.
func (f *FailableTB) Fatalf(format string, args ...any) {
	f.mu.Lock()
	if !f.failed {
		f.failed = true
		f.msg = fmt.Sprintf(format, args...)
		f.cancel()
	}
	goexit := f.goexit
	f.mu.Unlock()
	if goexit {
		runtime.Goexit()
	}
}

// Fatal records a failure and cancels the context. When [WithGoexit]
// is set, also terminates the goroutine via [runtime.Goexit].
func (f *FailableTB) Fatal(args ...any) {
	f.mu.Lock()
	if !f.failed {
		f.failed = true
		f.msg = fmt.Sprint(args...)
		f.cancel()
	}
	goexit := f.goexit
	f.mu.Unlock()
	if goexit {
		runtime.Goexit()
	}
}

// FailNow marks the test as failed. When [WithGoexit] is set, also
// terminates the goroutine via [runtime.Goexit].
func (f *FailableTB) FailNow() {
	f.Fail()
	f.mu.Lock()
	goexit := f.goexit
	f.mu.Unlock()
	if goexit {
		runtime.Goexit()
	}
}

// Fail marks the test as failed without terminating the goroutine.
func (f *FailableTB) Fail() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = true
}

// Errorf records a non-fatal error. Unlike [FailableTB.Fatal], it overwrites
// the message on each call and does not cancel the context.
func (f *FailableTB) Errorf(format string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = true
	f.msg = fmt.Sprintf(format, args...)
}

// Error records a non-fatal error.
func (f *FailableTB) Error(args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = true
	f.msg = fmt.Sprint(args...)
}

// Log appends a message to the log.
func (f *FailableTB) Log(args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logs = append(f.logs, fmt.Sprint(args...))
}

// Skip is a no-op on FailableTB. When [WithGoexit] is set,
// terminates the goroutine.
func (f *FailableTB) Skip(...any) {
	if f.goexit {
		runtime.Goexit()
	}
}

// Skipf is a no-op on FailableTB. When [WithGoexit] is set,
// terminates the goroutine.
func (f *FailableTB) Skipf(string, ...any) {
	if f.goexit {
		runtime.Goexit()
	}
}

// SkipNow is a no-op on FailableTB. When [WithGoexit] is set,
// terminates the goroutine.
func (f *FailableTB) SkipNow() {
	if f.goexit {
		runtime.Goexit()
	}
}

// Logf appends a formatted log message.
func (f *FailableTB) Logf(format string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logs = append(f.logs, fmt.Sprintf(format, args...))
}

// Context returns a [context.Context] that is cancelled when
// [FailableTB.Fatal] or [FailableTB.Fatalf] is called, matching the
// Go 1.24+ [testing.T.Context] lifecycle semantics.
func (f *FailableTB) Context() context.Context {
	return f.ctx
}

// Cleanup registers a function to be called when [FailableTB.RunCleanups]
// is invoked. Functions are called in LIFO order, matching [testing.T.Cleanup]
// semantics.
func (f *FailableTB) Cleanup(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cleanups = append(f.cleanups, fn)
}

// RunCleanups executes all registered cleanup functions in LIFO order,
// after joining every goroutine [FailableTB.Go] spawned. This simulates
// the cleanup phase that [testing.T] runs after a test completes. Use
// this to trigger [MethodStub.Verify] in tests that use [FailableTB]
// instead of a real [testing.T].
func (f *FailableTB) RunCleanups() {
	f.spawned.Wait()

	f.mu.Lock()
	fns := make([]func(), len(f.cleanups))
	copy(fns, f.cleanups)
	f.mu.Unlock()

	// LIFO order.
	for _, fn := range slices.Backward(fns) {
		fn()
	}
}

// Go runs fn in a goroutine whose panic fails this TB instead of
// crashing the process. In Go, a panic in a child goroutine bypasses
// every recover on the parent's stack and exits the process with the
// logs of the run half-written — so a planted defect or check body
// that spawns workers directly turns a red into a crash. Spawning
// through Go keeps the verdict: the panic is recorded as a failure,
// and [FailableTB.RunCleanups] joins every spawned goroutine before
// the cleanup phase reads it.
//
// The panic value is preserved in the failure message; the stack is
// not — a caller that needs the trace re-panics from its own recover.
func (f *FailableTB) Go(fn func()) {
	f.spawned.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				f.Errorf("goroutine panicked: %v", r)
			}
		}()
		fn()
	})
}

// TempDir is not supported on [FailableTB]. It always returns the empty
// string. Use a real [testing.T] when temporary directories are needed.
func (*FailableTB) TempDir() string {
	return ""
}

// Loop implements [BenchTB], returning true the number of times
// [FailableTB.WithIterations] allows and false thereafter.
//
// Unlike [testing.B.Loop] it neither times nor resets anything: what a caller
// checking a benchmark contract needs is a body that runs a known number of
// times, and the timing is [Contract]'s own.
func (f *FailableTB) Loop() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.looped >= f.iterations {
		return false
	}
	f.looped++
	return true
}

// Iterations returns how many times [FailableTB.Loop] returned true.
func (f *FailableTB) Iterations() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.looped
}

// ReportAllocs implements [BenchTB] by recording that allocation reporting was
// requested. There is no output to enable, so the record is the whole effect —
// it is what lets a caller check that a contract tracking allocations asked for
// them.
func (f *FailableTB) ReportAllocs() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.allocsReported = true
}

// AllocsReported reports whether [FailableTB.ReportAllocs] was called.
func (f *FailableTB) AllocsReported() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.allocsReported
}

// ReportMetric implements [BenchTB] by recording the value under its unit.
// A unit reported twice keeps the later value, matching [testing.B.ReportMetric].
func (f *FailableTB) ReportMetric(n float64, unit string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.metrics == nil {
		f.metrics = make(map[string]float64, 2)
	}
	f.metrics[unit] = n
}

// Metric returns the value recorded for unit, and whether one was recorded.
//
// The presence flag is the point: a latency contract reports `ns/op-p99` only
// when it is tracking latency, so "absent" and "zero" are different answers.
func (f *FailableTB) Metric(unit string) (float64, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.metrics[unit]
	return n, ok
}
