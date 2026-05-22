// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
)

func TestAssertNilSafe(t *testing.T) {
	t.Parallel()

	t.Run("passes when fn does not panic", func(t *testing.T) {
		t.Parallel()
		testkit.AssertNilSafe(t, func() {
			// no panic
		})
	})

	t.Run("fails when fn panics", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertNilSafe(f, func() {
			panic("boom")
		})
		testkit.True(t, f.Failed(), "must fail when fn panics")
		testkit.Assert(t, f.Msg()).Contains("panicked", "must describe the panic")
	})

	t.Run("passes when fn returns error", func(t *testing.T) {
		t.Parallel()
		testkit.AssertNilSafe(t, func() {
			_ = errors.New("expected error") //nolint:errcheck // testing that error return doesn't cause failure
		})
	})
}

func TestAssertNilCtx(t *testing.T) {
	t.Parallel()

	t.Run("passes when fn handles nil ctx", func(t *testing.T) {
		t.Parallel()
		testkit.AssertNilCtx(t, func(_ context.Context) error {
			return nil
		})
	})

	t.Run("fails when fn panics on nil ctx", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertNilCtx(f, func(ctx context.Context) error {
			_ = ctx.Err() // nil dereference
			return nil
		})
		testkit.True(t, f.Failed(), "must fail when fn panics on nil ctx")
	})
}

func TestAssertCtxDeadline(t *testing.T) {
	t.Parallel()

	t.Run("passes when fn returns deadline exceeded", func(t *testing.T) {
		t.Parallel()
		testkit.AssertCtxDeadline(t, func(ctx context.Context) error {
			return ctx.Err()
		})
	})

	t.Run("fails when fn returns nil", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertCtxDeadline(f, func(_ context.Context) error {
			return nil
		})
		testkit.True(t, f.Failed(), "must fail when fn returns nil")
	})

	t.Run("fails when fn returns non-context error", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertCtxDeadline(f, func(_ context.Context) error {
			return errors.New("unrelated")
		})
		testkit.True(t, f.Failed(), "must fail for non-context error")
	})
}

func TestAssertCtxCancellation(t *testing.T) {
	t.Parallel()

	t.Run("passes when fn returns context.Canceled", func(t *testing.T) {
		t.Parallel()
		testkit.AssertCtxCancellation(t, func(ctx context.Context) error {
			return ctx.Err()
		})
	})

	t.Run("passes when fn returns wrapped context.Canceled", func(t *testing.T) {
		t.Parallel()
		testkit.AssertCtxCancellation(t, func(ctx context.Context) error {
			return errors.Join(errors.New("op failed"), ctx.Err())
		})
	})

	t.Run("fails when fn returns nil", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertCtxCancellation(f, func(_ context.Context) error {
			return nil
		})
		testkit.True(t, f.Failed(), "must fail when fn returns nil")
	})

	t.Run("fails when fn returns non-context error", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertCtxCancellation(f, func(_ context.Context) error {
			return errors.New("not a context error")
		})
		testkit.True(t, f.Failed(), "must fail for non-context error")
	})
}

func TestAssertTimeout(t *testing.T) {
	t.Parallel()

	t.Run("passes when fn completes before deadline", func(t *testing.T) {
		t.Parallel()
		testkit.AssertTimeout(t, time.Second, func(_ context.Context) error {
			return nil
		})
	})

	t.Run("passes when fn returns non-deadline error", func(t *testing.T) {
		t.Parallel()
		testkit.AssertTimeout(t, time.Second, func(_ context.Context) error {
			return errors.New("some error")
		})
	})

	t.Run("fails when fn returns DeadlineExceeded", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertTimeout(f, 50*time.Millisecond, func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
		testkit.True(t, f.Failed(), "must fail when deadline exceeded")
		testkit.Assert(t, f.Msg()).Contains("timeout", "must describe the failure")
	})
}

func TestAssertPure(t *testing.T) {
	t.Parallel()

	t.Run("passes when state unchanged", func(t *testing.T) {
		t.Parallel()
		state := []string{"a", "b"}
		testkit.AssertPure(
			t,
			func() []string { return append([]string{}, state...) },
			func() { _ = len(state) }, // read-only
		)
	})

	t.Run("fails when state changed", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		state := []string{"a", "b"}
		testkit.AssertPure(
			f,
			func() []string { return append([]string{}, state...) },
			func() { state = append(state, "c") }, // mutation
		)
		testkit.True(t, f.Failed(), "must fail when state changed")
	})
}

func TestAssertBounded(t *testing.T) {
	t.Parallel()

	t.Run("passes when value in range", func(t *testing.T) {
		t.Parallel()
		testkit.AssertBounded(t, 0, 100, func() int { return 42 })
	})

	t.Run("passes at min boundary", func(t *testing.T) {
		t.Parallel()
		testkit.AssertBounded(t, 0, 100, func() int { return 0 })
	})

	t.Run("passes at max boundary", func(t *testing.T) {
		t.Parallel()
		testkit.AssertBounded(t, 0, 100, func() int { return 100 })
	})

	t.Run("fails below min", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertBounded(f, 10, 100, func() int { return 5 })
		testkit.True(t, f.Failed(), "must fail below min")
	})

	t.Run("fails above max", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertBounded(f, 0, 100, func() int { return 101 })
		testkit.True(t, f.Failed(), "must fail above max")
	})

	t.Run("works with strings", func(t *testing.T) {
		t.Parallel()
		testkit.AssertBounded(t, "a", "z", func() string { return "m" })
	})

	t.Run("works with float64", func(t *testing.T) {
		t.Parallel()
		testkit.AssertBounded(t, 0.0, 1.0, func() float64 { return 0.5 })
	})
}
