// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"runtime"
	"sync"
	"time"
)

// TestClock is a deterministic, manually-advanced clock for testing. Time
// does not advance on its own — only [TestClock.Advance] moves the clock
// forward. Goroutines blocked in [TestClock.Sleep], [TestClock.After], or
// waiting on a [Timer] are woken when virtual time reaches their deadline.
//
//	clk := testkit.NewTestClock(time.Unix(0, 0))
//	clk.Advance(5 * time.Second)
//	clk.Now() // time.Unix(5, 0)
type TestClock struct {
	mu      sync.Mutex
	cond    *sync.Cond
	now     time.Time
	waiters []*testWaiter
}

// testWaiter represents a goroutine blocked on a virtual-time deadline.
type testWaiter struct {
	deadline time.Time
	ch       chan time.Time
	once     sync.Once
}

func (w *testWaiter) fire(at time.Time) {
	w.once.Do(func() {
		w.ch <- at
	})
}

// NewTestClock returns a [TestClock] initialized at the given origin time.
func NewTestClock(origin time.Time) *TestClock {
	tc := &TestClock{now: origin}
	tc.cond = sync.NewCond(&tc.mu)
	return tc
}

// Now returns the current virtual time.
func (c *TestClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves virtual time forward by d and wakes all waiters whose
// deadline has been reached. Panics if d is negative.
func (c *TestClock) Advance(d time.Duration) {
	if d < 0 {
		panic("testkit.TestClock.Advance: negative duration") //nolint:forbidigo // intentional panic on misuse
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	c.fireWaiters()
	c.cond.Broadcast()
}

// AwaitWaiters spins until at least n goroutines are blocked in [TestClock.Sleep],
// [TestClock.After], or on a [Timer]. Use this instead of time.Sleep for
// deterministic goroutine synchronization in tests:
//
//	go func() { clk.Sleep(5 * time.Second) }()
//	clk.AwaitWaiters(1)   // deterministic — no real-time dependency
//	clk.Advance(6 * time.Second)
func (c *TestClock) AwaitWaiters(n int) {
	for {
		c.mu.Lock()
		count := len(c.waiters)
		c.mu.Unlock()
		if count >= n {
			return
		}
		runtime.Gosched()
	}
}

// Sleep blocks until virtual time reaches now + d. With a [TestClock],
// this does not block real wall-clock time — [Advance] wakes the sleeper.
func (c *TestClock) Sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	c.mu.Lock()
	deadline := c.now.Add(d)
	w := &testWaiter{deadline: deadline, ch: make(chan time.Time, 1)}
	c.waiters = append(c.waiters, w)
	c.mu.Unlock()
	<-w.ch
}

// After returns a channel that receives the deadline time when virtual
// time reaches now + d.
func (c *TestClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	deadline := c.now.Add(d)
	w := &testWaiter{deadline: deadline, ch: make(chan time.Time, 1)}
	if !c.now.Before(deadline) {
		// Already past deadline.
		w.fire(deadline)
	} else {
		c.waiters = append(c.waiters, w)
	}
	return w.ch
}

// NewTimer creates a [Timer] that fires when virtual time reaches now + d.
func (c *TestClock) NewTimer(d time.Duration) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	deadline := c.now.Add(d)
	w := &testWaiter{deadline: deadline, ch: make(chan time.Time, 1)}
	t := &testTimer{clock: c, waiter: w}
	if !c.now.Before(deadline) {
		w.fire(deadline)
		t.fired = true
	} else {
		c.waiters = append(c.waiters, w)
	}
	return t
}

// fireWaiters fires all waiters whose deadline is <= c.now. Must be called
// with c.mu held.
func (c *TestClock) fireWaiters() {
	remaining := c.waiters[:0]
	for _, w := range c.waiters {
		if !c.now.Before(w.deadline) {
			w.fire(w.deadline)
		} else {
			remaining = append(remaining, w)
		}
	}
	c.waiters = remaining
}

// removeWaiter removes a waiter from the pending list. Returns true if
// the waiter was found and removed (meaning it hadn't fired yet).
// Must be called with c.mu held.
func (c *TestClock) removeWaiter(w *testWaiter) bool {
	for i, existing := range c.waiters {
		if existing == w {
			c.waiters = append(c.waiters[:i], c.waiters[i+1:]...)
			return true
		}
	}
	return false
}

// testTimer implements [Timer] for [TestClock].
type testTimer struct {
	clock   *TestClock
	waiter  *testWaiter
	mu      sync.Mutex
	fired   bool
	stopped bool
}

func (t *testTimer) C() <-chan time.Time {
	return t.waiter.ch
}

func (t *testTimer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return false
	}
	t.clock.mu.Lock()
	removed := t.clock.removeWaiter(t.waiter)
	t.clock.mu.Unlock()
	t.stopped = true
	// removed is true only if the waiter was still pending (not yet fired).
	return removed
}

func (t *testTimer) Reset(d time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.clock.mu.Lock()
	wasActive := t.clock.removeWaiter(t.waiter)

	// Drain any leftover buffered value from a prior fire. Without this,
	// Reset on a fired-but-undrained timer would deadlock when the new
	// waiter's fire() tries to send on a full channel while holding
	// t.clock.mu.
	select {
	case <-t.waiter.ch:
	default:
	}

	// Create new waiter with updated deadline. Reuse the channel so
	// callers holding C() still receive from the right channel.
	deadline := t.clock.now.Add(d)
	w := &testWaiter{deadline: deadline, ch: t.waiter.ch}
	t.waiter = w
	t.stopped = false

	if !t.clock.now.Before(deadline) {
		w.fire(deadline)
	} else {
		t.clock.waiters = append(t.clock.waiters, w)
	}
	t.clock.mu.Unlock()

	return wasActive
}
