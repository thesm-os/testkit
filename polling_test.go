// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"sync/atomic"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
)

func TestRetryUntil(t *testing.T) {
	t.Parallel()

	t.Run("succeeds when pred returns true immediately", func(t *testing.T) {
		t.Parallel()
		testkit.RetryUntil(t, time.Second, func() bool { return true }, "must pass")
	})

	t.Run("retries until pred returns true", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int64
		testkit.RetryUntil(t, time.Second, func() bool {
			return calls.Add(1) >= 3
		}, "must succeed after 3 calls")
		testkit.True(t, calls.Load() >= 3, "must have retried")
	})

	t.Run("fatals on timeout", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.RetryUntil(f, 50*time.Millisecond, func() bool { return false }, "never succeeds")
		testkit.True(t, f.Failed(), "must fatal on timeout")
	})
}

func TestAssertEventually(t *testing.T) {
	t.Parallel()

	t.Run("passes when assertion succeeds immediately", func(t *testing.T) {
		t.Parallel()
		testkit.AssertEventually(t, time.Second, 10*time.Millisecond, func(tb testing.TB) {
			tb.Helper()
			testkit.True(tb, true, "always true")
		}, "must pass")
	})

	t.Run("retries until assertion passes", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int64
		testkit.AssertEventually(t, time.Second, 10*time.Millisecond, func(tb testing.TB) {
			tb.Helper()
			n := calls.Add(1)
			testkit.True(tb, n >= 3, "need at least 3 calls")
		}, "must converge")
	})

	t.Run("fatals on timeout with last failure message", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertEventually(f, 50*time.Millisecond, 10*time.Millisecond, func(tb testing.TB) {
			tb.Helper()
			testkit.True(tb, false, "always fails")
		}, "convergence")
		testkit.True(t, f.Failed(), "must fatal on timeout")
		testkit.Assert(t, f.Msg()).
			Contains("convergence", "must include context message").
			Contains("always fails", "must include last failure")
	})
}
