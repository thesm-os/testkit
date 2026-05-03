// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package polling

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
)

// RetryUntil calls pred repeatedly with exponential backoff until it returns
// true or timeout expires. Starts at 1ms and doubles each attempt, capped at
// timeout/4. Calls tb.Fatalf on timeout.
//
//	testkit.RetryUntil(t, 5*time.Second, func() bool {
//	    return cache.Contains(key)
//	}, "key must appear in cache")
func RetryUntil(tb testing.TB, timeout time.Duration, pred func() bool, msg string) {
	tb.Helper()
	deadline := time.Now().Add(timeout)
	interval := time.Millisecond
	maxInterval := max(timeout/4, time.Millisecond)
	for {
		if pred() {
			return
		}
		if time.Now().After(deadline) {
			tb.Fatalf("%s: timed out after %v", msg, timeout)
			return
		}
		time.Sleep(interval)
		interval *= 2
		if interval > maxInterval {
			interval = maxInterval
		}
	}
}

// AssertEventually calls fn repeatedly at the given interval until it passes
// (does not call tb.Fatal/tb.Fatalf) or until timeout expires. On timeout,
// it reports the last assertion failure. This is stronger than [RetryUntil]
// because fn can use the full assertion API.
//
//	testkit.AssertEventually(t, 5*time.Second, 100*time.Millisecond, func(t testing.TB) {
//	    testkit.Equal(t, cache.Get(key), want, "cached value must match")
//	}, "cache must converge")
func AssertEventually(tb testing.TB, timeout, interval time.Duration, fn func(tb testing.TB), msg string) {
	tb.Helper()
	deadline := time.Now().Add(timeout)
	for {
		f := testkit.NewFailableTB()
		fn(f)
		if !f.Failed() {
			return
		}
		if time.Now().After(deadline) {
			tb.Fatalf("%s: timed out after %v — last failure: %s", msg, timeout, f.Msg())
			return
		}
		time.Sleep(interval)
	}
}
