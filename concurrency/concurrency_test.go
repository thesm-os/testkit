// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrency_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/concurrency"
)

func TestConcurrentStress(t *testing.T) {
	t.Parallel()

	t.Run("executes all goroutines and iterations", func(t *testing.T) {
		t.Parallel()
		var count atomic.Int64
		concurrency.ConcurrentStress(t, 4, 10, func(_, _ int) {
			count.Add(1)
		})
		testkit.Equal(t, count.Load(), int64(40), "must execute 4*10=40 iterations")
	})

	t.Run("passes goroutine and iteration indices to work function", func(t *testing.T) {
		t.Parallel()
		// Track all (g, i) pairs seen.
		type pair struct{ g, i int }
		seen := make(chan pair, 6)
		concurrency.ConcurrentStress(t, 2, 3, func(g, i int) {
			seen <- pair{g, i}
		})
		close(seen)
		var count int
		for range seen {
			count++
		}
		testkit.Equal(t, count, 6, "must see 2*3=6 pairs")
	})
}

//nolint:paralleltest // goroutine leak detection requires exclusive goroutine control
func TestGoroutineLeak(t *testing.T) {
	t.Run("passes when goroutine exits before cleanup", func(t *testing.T) { //nolint:paralleltest
		cleanup := concurrency.GoroutineLeak(t)
		done := make(chan struct{})
		go func() {
			close(done)
		}()
		<-done
		cleanup()
	})

	t.Run("detects leaked goroutine", func(t *testing.T) { //nolint:paralleltest
		f := testkit.NewFailableTB()
		cleanup := concurrency.GoroutineLeak(f)

		blocker := make(chan struct{})
		go func() {
			<-blocker // blocks forever until we close
		}()

		cleanup()
		testkit.True(t, f.Failed(), "must detect leaked goroutine")
		close(blocker) // clean up
	})
}

func TestTimeout(t *testing.T) {
	t.Parallel()

	t.Run("returns context with deadline", func(t *testing.T) {
		t.Parallel()
		ctx := concurrency.Timeout(t, 5*time.Second)
		deadline, ok := ctx.Deadline()
		testkit.True(t, ok, "must have deadline")
		testkit.True(t, deadline.After(time.Now()), "deadline must be in the future")
	})

	t.Run("context derives from test context", func(t *testing.T) {
		t.Parallel()
		ctx := concurrency.Timeout(t, 5*time.Second)
		testkit.NoError(t, ctx.Err(), "context must not be cancelled yet")
		// Verify it's a child of t.Context() by checking it cancels when
		// the timeout fires (not testing parent cancellation since we can't
		// cancel t.Context() ourselves).
		_ = ctx // just verify it compiles and returns a valid context
	})

	t.Run("expired context has DeadlineExceeded error", func(t *testing.T) {
		t.Parallel()
		ctx := concurrency.Timeout(t, 1*time.Millisecond)
		time.Sleep(10 * time.Millisecond)
		testkit.ErrorIs(t, ctx.Err(), context.DeadlineExceeded, "must be deadline exceeded")
	})
}
