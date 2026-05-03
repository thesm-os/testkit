// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
)

func TestRealClock(t *testing.T) {
	t.Parallel()

	t.Run("Now returns current time", func(t *testing.T) {
		t.Parallel()
		clk := testkit.RealClock()
		before := time.Now()
		got := clk.Now()
		after := time.Now()
		testkit.True(t, !got.Before(before) && !got.After(after),
			"Now must return current wall-clock time")
	})

	t.Run("Sleep blocks briefly", func(t *testing.T) {
		t.Parallel()
		clk := testkit.RealClock()
		start := time.Now()
		clk.Sleep(5 * time.Millisecond)
		testkit.True(t, time.Since(start) >= 5*time.Millisecond,
			"Sleep must block for at least the requested duration")
	})

	t.Run("After sends time", func(t *testing.T) {
		t.Parallel()
		clk := testkit.RealClock()
		ch := clk.After(5 * time.Millisecond)
		select {
		case got := <-ch:
			testkit.True(t, !got.IsZero(), "After must send non-zero time")
		case <-time.After(time.Second):
			t.Fatal("After must fire")
		}
	})

	t.Run("NewTimer fires", func(t *testing.T) {
		t.Parallel()
		clk := testkit.RealClock()
		timer := clk.NewTimer(10 * time.Millisecond)
		select {
		case <-timer.C():
			// Success.
		case <-time.After(time.Second):
			t.Fatal("real timer must fire")
		}
	})

	t.Run("Timer.Stop prevents firing", func(t *testing.T) {
		t.Parallel()
		clk := testkit.RealClock()
		timer := clk.NewTimer(time.Hour)
		stopped := timer.Stop()
		testkit.True(t, stopped, "must stop active timer")
	})

	t.Run("Timer.Reset reschedules", func(t *testing.T) {
		t.Parallel()
		clk := testkit.RealClock()
		timer := clk.NewTimer(time.Hour)
		timer.Reset(5 * time.Millisecond)
		select {
		case <-timer.C():
			// Success.
		case <-time.After(time.Second):
			t.Fatal("reset timer must fire")
		}
	})
}
