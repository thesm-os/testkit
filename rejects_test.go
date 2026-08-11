// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"runtime"
	"testing"

	"go.thesmos.sh/testkit"
)

// Rejects is the assertion that an assertion can fail, so its own test is the
// same question asked twice: it has to pass on a check that rejects, and fail
// on one that does not.
func TestRejects(t *testing.T) {
	t.Parallel()

	t.Run("passes when the check fails", func(t *testing.T) {
		t.Parallel()
		testkit.Rejects(t, "a check that fails is one that can fail", func(tb testing.TB) {
			tb.Helper()
			testkit.Equal(tb, 1, 2, "one is not two")
		})
	})

	t.Run("returns why the check failed", func(t *testing.T) {
		t.Parallel()
		// The message is what separates "something went wrong" from "the thing
		// this check is about went wrong", and a guard that cannot tell them
		// apart is the defect it exists to catch, one level up.
		got := testkit.Rejects(t, "the check fails", func(tb testing.TB) {
			tb.Helper()
			testkit.Equal(tb, 1, 2, "one is not two")
		})
		testkit.Assert(t, got).Contains("one is not two", "carrying the check's own message")
	})

	t.Run("fails when the check passes", func(t *testing.T) {
		t.Parallel()
		// The whole point, and the case a corpus check that only calls NoError
		// lands in. Driven through a FailableTB rather than asserted about,
		// because the failure being observed is Rejects's own.
		f := testkit.NewFailableTB()
		testkit.Rejects(f, "a vacuous check", func(tb testing.TB) {
			tb.Helper()
			testkit.Equal(tb, 1, 1, "one is one")
		})
		testkit.True(t, f.Failed(), "a check that cannot fail must be reported")
		testkit.Assert(t, f.Msg()).Contains("an implementation it must reject",
			"and the message says what was expected of it")
	})

	t.Run("stops the check at its first failed assertion", func(t *testing.T) {
		t.Parallel()
		// Goexit semantics, which is what a check written against *testing.T
		// relies on: everything after a Fatal is written assuming it did not
		// run, and a stand-in that let it continue would report a second
		// failure the real harness never sees.
		reached := false
		testkit.Rejects(t, "the check fails at its first assertion", func(tb testing.TB) {
			tb.Helper()
			testkit.Equal(tb, 1, 2, "one is not two")
			reached = true
		})
		testkit.False(t, reached, "nothing after the failed assertion runs")
	})

	t.Run("leaves no goroutine behind", func(t *testing.T) {
		t.Parallel()
		// The goroutine exists only because runtime.Goexit needs one to exit.
		// It has to be joined before the call returns, or every check driven
		// through this leaks one and a consumer's leak detector reports it
		// against their code.
		defer goroutineLeakGuard(t)()
		testkit.Rejects(t, "the check fails", func(tb testing.TB) {
			tb.Helper()
			testkit.Equal(tb, 1, 2, "one is not two")
		})
	})
}

// goroutineLeakGuard is [concurrency.GoroutineLeak] restated, because the root
// package cannot import a package that imports it.
//
// Same claim, smaller: the count when the guard is installed has to be the
// count when it runs.
func goroutineLeakGuard(tb testing.TB) func() {
	tb.Helper()
	before := runtime.NumGoroutine()
	return func() {
		tb.Helper()
		if after := runtime.NumGoroutine(); after > before {
			tb.Errorf("goroutines leaked: %d before, %d after", before, after)
		}
	}
}
