// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"cmp"
	"context"
	"errors"
	"testing"
	"time"
)

// AssertNilSafe calls fn and asserts it does not panic. The function may
// return an error — that's expected. It must not crash. Use this to verify
// that methods handle zero-value or nil inputs gracefully.
//
//	testkit.AssertNilSafe(t, func() {
//	    _ = store.Put(ctx, Item{})
//	})
func AssertNilSafe(tb testing.TB, fn func()) {
	tb.Helper()
	defer func() {
		if r := recover(); r != nil {
			tb.Errorf("nilsafe: function panicked: %v", r)
		}
	}()
	fn()
}

// AssertNilCtx calls fn with a nil context and asserts it does not panic.
// The function may return an error — that's expected. It must not crash.
// Methods should check ctx != nil or use ctx.Err() defensively.
func AssertNilCtx(tb testing.TB, fn func(ctx context.Context) error) {
	tb.Helper()
	defer func() {
		if r := recover(); r != nil {
			tb.Errorf("nil context: function panicked: %v", r)
		}
	}()
	_ = fn(nil) //nolint:staticcheck // intentionally passing nil context to test robustness
}

// AssertCtxCancellation calls fn with an already-cancelled context and asserts
// the returned error wraps [context.Canceled] or [context.DeadlineExceeded].
// Methods that respect context cancellation should return promptly with a
// context error.
//
//	testkit.AssertCtxCancellation(t, func(ctx context.Context) error {
//	    _, err := store.Get(ctx, "id")
//	    return err
//	})
func AssertCtxCancellation(tb testing.TB, fn func(ctx context.Context) error) {
	tb.Helper()
	ctx, cancel := context.WithCancel(tb.Context())
	cancel()
	err := fn(ctx)
	if err == nil {
		tb.Errorf("ctx cancellation: expected error from cancelled context, got nil")
		return
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		tb.Errorf("ctx cancellation: expected context.Canceled or context.DeadlineExceeded, got %v", err)
	}
}

// AssertCtxDeadline calls fn with an already-expired deadline context and
// asserts the returned error wraps [context.DeadlineExceeded]. Methods that
// respect context deadlines should return promptly with a deadline error.
//
//	testkit.AssertCtxDeadline(t, func(ctx context.Context) error {
//	    _, err := store.Get(ctx, "id")
//	    return err
//	})
func AssertCtxDeadline(tb testing.TB, fn func(ctx context.Context) error) {
	tb.Helper()
	ctx, cancel := context.WithDeadline(tb.Context(), time.Now().Add(-time.Second))
	defer cancel()
	err := fn(ctx)
	if err == nil {
		tb.Errorf("ctx deadline: expected error from expired deadline, got nil")
		return
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		tb.Errorf("ctx deadline: expected context.DeadlineExceeded, got %v", err)
	}
}

// AssertTimeout calls fn with a context that has the given deadline and
// asserts fn returns before the deadline expires. If fn returns
// [context.DeadlineExceeded], the method did not honor the deadline — the
// test fails.
//
//	testkit.AssertTimeout(t, 5*time.Second, func(ctx context.Context) error {
//	    return runner.Run(ctx)
//	})
func AssertTimeout(tb testing.TB, deadline time.Duration, fn func(ctx context.Context) error) {
	tb.Helper()
	ctx, cancel := context.WithTimeout(tb.Context(), deadline)
	defer cancel()
	err := fn(ctx)
	if errors.Is(err, context.DeadlineExceeded) {
		tb.Errorf("timeout: function did not complete within %v", deadline)
	}
}

// AssertPure calls observe before and after fn, then asserts the observable
// state did not change. Use this to verify that read-only methods have no
// side effects on the observable state.
//
// Observers must return values comparable via [cmp.Equal]. For types with
// unexported fields, return a deep-copied projection of the public fields.
// For non-deterministic state (timestamps, random IDs), exclude from the
// projection.
//
//	testkit.AssertPure(t,
//	    func() []Item { return store.List(ctx) },
//	    func() { _, _ = store.Get(ctx, "id") },
//	)
func AssertPure[S any](tb testing.TB, observe func() S, fn func()) {
	tb.Helper()
	before := observe()
	fn()
	after := observe()
	Equal(tb, before, after, "pure: state must not change")
}

// AssertBounded calls fn and asserts the result is in [min, max] (inclusive).
//
//	testkit.AssertBounded(t, 0, 1000, func() int {
//	    return counter.Count(ctx)
//	})
func AssertBounded[T cmp.Ordered](tb testing.TB, lower, upper T, fn func() T) {
	tb.Helper()
	got := fn()
	if got < lower || got > upper {
		tb.Errorf("bounded: got %v, want [%v, %v]", got, lower, upper)
	}
}
