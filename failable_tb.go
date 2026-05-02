// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// FailableTB is a [testing.TB] stub that captures the first fatal message
// without aborting the test process. Use it to verify that assertion helpers
// produce the expected failure output without actually failing the parent test.
//
//	f := testkit.NewFailableTB()
//	testkit.Equal(f, 1, 2, "must match")
//	if !f.Failed() { t.Fatal("Equal should have failed") }
type FailableTB struct {
	testing.TB // embedded nil interface — panics on unimplemented methods

	mu          sync.Mutex
	ctx         context.Context //nolint:containedctx // test double must hold ctx to implement testing.TB.Context
	cancel      context.CancelFunc
	name        string
	msg         string
	logs        []string
	helperCalls int
	failed      bool
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

// Fatalf records a failure and cancels the context. Subsequent calls are
// ignored — the first failure wins.
func (f *FailableTB) Fatalf(format string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failed {
		return
	}
	f.failed = true
	f.msg = fmt.Sprintf(format, args...)
	f.cancel()
}

// Fatal records a failure and cancels the context. Subsequent calls are
// ignored — the first failure wins.
func (f *FailableTB) Fatal(args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failed {
		return
	}
	f.failed = true
	f.msg = fmt.Sprint(args...)
	f.cancel()
}

// Errorf records a non-fatal error. Unlike [FailableTB.Fatal], it overwrites
// the message on each call and does not cancel the context.
func (f *FailableTB) Errorf(format string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = true
	f.msg = fmt.Sprintf(format, args...)
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

// Cleanup is a no-op. Present to satisfy the [testing.TB] interface.
func (*FailableTB) Cleanup(func()) {}

// TempDir is not supported on [FailableTB]. It always returns the empty
// string. Use a real [testing.T] when temporary directories are needed.
func (*FailableTB) TempDir() string {
	return ""
}
