// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package clock_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
)

func TestTestClock(t *testing.T) {
	t.Parallel()

	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("Now returns origin initially", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		testkit.Equal(t, clk.Now(), origin, "initial time must be origin")
	})

	t.Run("Advance moves time forward", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		clk.Advance(5 * time.Second)
		testkit.Equal(t, clk.Now(), origin.Add(5*time.Second), "must advance by 5s")
	})

	t.Run("multiple Advance calls accumulate", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		clk.Advance(3 * time.Second)
		clk.Advance(2 * time.Second)
		testkit.Equal(t, clk.Now(), origin.Add(5*time.Second), "must accumulate advances")
	})

	t.Run("Advance panics on negative duration", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		defer func() {
			r := recover()
			testkit.True(t, r != nil, "must panic on negative Advance")
		}()
		clk.Advance(-1 * time.Second)
	})

	t.Run("AwaitWaiters returns when waiters registered", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		go func() {
			clk.Sleep(5 * time.Second)
		}()
		clk.AwaitWaiters(1)
		testkit.True(t, true, "AwaitWaiters must return once waiter registered")
		clk.Advance(6 * time.Second) // unblock the goroutine
	})

	t.Run("Sleep unblocks when time advances past deadline", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		done := make(chan struct{})
		go func() {
			clk.Sleep(5 * time.Second)
			close(done)
		}()

		clk.AwaitWaiters(1)
		clk.Advance(6 * time.Second)

		select {
		case <-done:
			// Success — Sleep unblocked.
		case <-time.After(time.Second):
			t.Fatal("Sleep did not unblock after Advance")
		}
	})

	t.Run("Sleep with zero duration returns immediately", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		done := make(chan struct{})
		go func() {
			clk.Sleep(0)
			close(done)
		}()
		select {
		case <-done:
			// Success.
		case <-time.After(time.Second):
			t.Fatal("Sleep(0) must return immediately")
		}
	})

	t.Run("Sleep with negative duration returns immediately", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		done := make(chan struct{})
		go func() {
			clk.Sleep(-1 * time.Second)
			close(done)
		}()
		select {
		case <-done:
			// Success.
		case <-time.After(time.Second):
			t.Fatal("Sleep with negative duration must return immediately")
		}
	})

	t.Run("Sleep does not block real wall-clock time", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)

		done := make(chan struct{})
		go func() {
			clk.Sleep(1 * time.Hour)
			close(done)
		}()

		clk.AwaitWaiters(1)
		start := time.Now()
		clk.Advance(2 * time.Hour)
		<-done
		elapsed := time.Since(start)
		testkit.True(t, elapsed < 100*time.Millisecond,
			"virtual Sleep must not block real wall-clock time")
	})

	t.Run("multiple sleepers wake", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)

		ch1 := make(chan struct{})
		ch2 := make(chan struct{})

		go func() {
			clk.Sleep(3 * time.Second)
			close(ch1)
		}()
		go func() {
			clk.Sleep(1 * time.Second)
			close(ch2)
		}()

		clk.AwaitWaiters(2)
		clk.Advance(5 * time.Second)

		select {
		case <-ch1:
		case <-time.After(time.Second):
			t.Fatal("sleeper 1 did not wake")
		}
		select {
		case <-ch2:
		case <-time.After(time.Second):
			t.Fatal("sleeper 2 did not wake")
		}
	})

	t.Run("After sends time when deadline reached", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		ch := clk.After(3 * time.Second)

		clk.Advance(4 * time.Second)

		select {
		case got := <-ch:
			testkit.Equal(t, got, origin.Add(3*time.Second),
				"After must send the deadline time")
		case <-time.After(time.Second):
			t.Fatal("After channel did not fire")
		}
	})

	t.Run("After with zero duration fires immediately", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		ch := clk.After(0)
		select {
		case <-ch:
			// Success — fired immediately.
		case <-time.After(time.Second):
			t.Fatal("After(0) must fire immediately")
		}
	})

	t.Run("NewTimer fires when deadline reached", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		timer := clk.NewTimer(5 * time.Second)

		clk.Advance(6 * time.Second)

		select {
		case got := <-timer.C():
			testkit.Equal(t, got, origin.Add(5*time.Second),
				"timer must fire with deadline time")
		case <-time.After(time.Second):
			t.Fatal("timer did not fire")
		}
	})

	t.Run("NewTimer with zero duration fires immediately", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		timer := clk.NewTimer(0)
		select {
		case <-timer.C():
			// Success.
		case <-time.After(time.Second):
			t.Fatal("NewTimer(0) must fire immediately")
		}
	})

	t.Run("Timer.Stop prevents firing", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		timer := clk.NewTimer(5 * time.Second)

		stopped := timer.Stop()
		testkit.True(t, stopped, "Stop must return true for active timer")

		clk.Advance(10 * time.Second)

		select {
		case <-timer.C():
			t.Fatal("stopped timer must not fire")
		case <-time.After(50 * time.Millisecond):
			// Success — timer did not fire.
		}
	})

	t.Run("Timer.Stop returns false after firing", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		timer := clk.NewTimer(1 * time.Second)
		clk.Advance(2 * time.Second)
		<-timer.C()

		stopped := timer.Stop()
		testkit.False(t, stopped, "Stop must return false after timer fired")
	})

	t.Run("Timer.Stop on already-stopped timer returns false", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		timer := clk.NewTimer(5 * time.Second)
		timer.Stop()
		testkit.False(t, timer.Stop(), "second Stop must return false")
	})

	t.Run("Timer.Reset reschedules", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		timer := clk.NewTimer(10 * time.Second)

		timer.Reset(2 * time.Second)
		clk.Advance(3 * time.Second)

		select {
		case got := <-timer.C():
			testkit.True(t, !got.IsZero(), "timer must fire after reset")
		case <-time.After(time.Second):
			t.Fatal("reset timer did not fire")
		}
	})

	t.Run("Timer.Reset on fired timer", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		timer := clk.NewTimer(1 * time.Second)
		clk.Advance(2 * time.Second)
		<-timer.C()

		wasActive := timer.Reset(3 * time.Second)
		testkit.False(t, wasActive, "Reset after fire must return false")

		clk.Advance(4 * time.Second)
		select {
		case <-timer.C():
			// Success — reset timer fires.
		case <-time.After(time.Second):
			t.Fatal("reset timer must fire")
		}
	})
}

// A virtual clock that dropped unfired waiters would silently lose scheduled
// work — the exact failure mode a test clock exists to prevent.
func TestTestClockWaiterLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("advancing past one deadline leaves later waiters armed", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(time.Unix(0, 0))
		soon := clk.After(1 * time.Second)
		later := clk.After(10 * time.Second)

		clk.Advance(2 * time.Second)

		select {
		case <-soon:
		default:
			t.Fatal("the elapsed waiter must have fired")
		}
		select {
		case <-later:
			t.Fatal("the future waiter must not have fired")
		default:
		}
	})

	t.Run("a retained waiter still fires once its deadline passes", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(time.Unix(0, 0))
		later := clk.After(10 * time.Second)

		clk.Advance(2 * time.Second)
		clk.Advance(10 * time.Second)

		select {
		case <-later:
		default:
			t.Fatal("the retained waiter must fire after its deadline")
		}
	})
}

// Reset re-arms a timer against the current virtual now. A future deadline
// goes back on the waiter list; a past one fires immediately.
func TestTestTimerReset(t *testing.T) {
	t.Parallel()

	t.Run("reports whether the timer was pending", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(time.Unix(0, 0))
		timer := clk.NewTimer(1 * time.Second)
		testkit.True(t, timer.Reset(5*time.Second), "a pending timer reports active")
	})

	t.Run("a future deadline does not fire early", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(time.Unix(0, 0))
		timer := clk.NewTimer(1 * time.Second)
		timer.Reset(5 * time.Second)

		clk.Advance(2 * time.Second)
		select {
		case <-timer.C():
			t.Fatal("must not fire before the new deadline")
		default:
		}
	})

	t.Run("a future deadline fires once reached", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(time.Unix(0, 0))
		timer := clk.NewTimer(1 * time.Second)
		timer.Reset(5 * time.Second)

		clk.Advance(6 * time.Second)
		select {
		case <-timer.C():
		default:
			t.Fatal("must fire once the new deadline passes")
		}
	})

	// Reset on a fired-but-undrained timer must discard the stale value.
	// The channel is buffered to one, so leaving it full would deadlock the
	// next fire() while the clock lock is held.
	t.Run("a stale buffered fire is discarded", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(time.Unix(0, 0))
		timer := clk.NewTimer(1 * time.Second)
		clk.Advance(2 * time.Second) // fires; nobody reads C()

		testkit.False(t, timer.Reset(5*time.Second), "a fired timer is no longer pending")
		select {
		case <-timer.C():
			t.Fatal("the stale value must be drained, not delivered")
		default:
		}

		clk.Advance(6 * time.Second)
		select {
		case <-timer.C():
		default:
			t.Fatal("the reset timer must still fire on its new deadline")
		}
	})

	t.Run("a non-positive deadline fires immediately", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(time.Unix(0, 0))
		timer := clk.NewTimer(1 * time.Second)
		timer.Reset(0)

		select {
		case <-timer.C():
		default:
			t.Fatal("Reset(0) leaves the deadline at now, which is already reached")
		}
	})
}
