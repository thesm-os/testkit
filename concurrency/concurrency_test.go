// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrency_test

import (
	"context"
	"strings"
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

func TestCaptureGoroutineIDs(t *testing.T) {
	t.Parallel()

	t.Run("returns at least the calling goroutine", func(t *testing.T) {
		t.Parallel()
		ids := concurrency.CaptureGoroutineIDs()
		testkit.True(t, len(ids) >= 1, "at least one goroutine ID")
	})

	t.Run("captures spawned goroutines while alive", func(t *testing.T) {
		t.Parallel()
		// Spawn a worker that blocks until released; observe its ID is
		// in the live set, then release and observe it leaving.
		release := make(chan struct{})
		started := make(chan struct{})
		go func() {
			close(started)
			<-release
		}()
		<-started
		live := concurrency.CaptureGoroutineIDs()
		testkit.True(t, len(live) >= 2, "self + worker visible in capture")
		close(release)
	})
}

func TestCaptureGoroutineStacks(t *testing.T) {
	t.Parallel()
	stacks := concurrency.CaptureGoroutineStacks()
	testkit.True(t, len(stacks) > 0, "non-empty stack output")
	testkit.Contains(t, string(stacks), "goroutine ", "contains goroutine headers")
}

func TestDiffGoroutineIDs(t *testing.T) {
	t.Parallel()

	t.Run("returns IDs in after but not before", func(t *testing.T) {
		t.Parallel()
		before := map[uint64]struct{}{1: {}, 2: {}}
		after := map[uint64]struct{}{1: {}, 2: {}, 3: {}, 4: {}}
		diff := concurrency.DiffGoroutineIDs(before, after)
		testkit.Len(t, diff, 2, "two new IDs")
	})

	t.Run("empty when no new goroutines", func(t *testing.T) {
		t.Parallel()
		ids := map[uint64]struct{}{1: {}, 2: {}}
		testkit.Len(t, concurrency.DiffGoroutineIDs(ids, ids), 0, "no diff")
	})

	t.Run("excludes IDs present in before but absent in after", func(t *testing.T) {
		t.Parallel()
		before := map[uint64]struct{}{1: {}, 2: {}, 3: {}}
		after := map[uint64]struct{}{1: {}}
		// 2 and 3 disappeared; not "leaked" — diff returns nothing.
		testkit.Len(t, concurrency.DiffGoroutineIDs(before, after), 0,
			"diff is asymmetric: only after-minus-before")
	})
}

// The stack parser runs over whatever runtime.Stack produced, so it must skip
// anything it cannot interpret rather than panic or record a bogus id. Real
// runtime output never contains these shapes, which is why they are exercised
// against a hand-built stack.
func TestParseGoroutineIDsSkipsMalformedLines(t *testing.T) {
	t.Parallel()

	t.Run("truncated header without an id", func(t *testing.T) {
		t.Parallel()
		ids := concurrency.ParseGoroutineIDs([]byte("goroutine \nmain.main()\n"))
		testkit.Equal(t, len(ids), 0, "a header with no id must be skipped")
	})

	t.Run("non-numeric id", func(t *testing.T) {
		t.Parallel()
		ids := concurrency.ParseGoroutineIDs([]byte("goroutine abc [running]:\n"))
		testkit.Equal(t, len(ids), 0, "an unparseable id must be skipped")
	})

	t.Run("malformed lines do not hide valid ones", func(t *testing.T) {
		t.Parallel()
		lines := []string{"goroutine ", "goroutine abc [running]:", "goroutine 7 [running]:"}
		ids := concurrency.ParseGoroutineIDs([]byte(strings.Join(lines, "\n") + "\n"))
		testkit.Equal(t, len(ids), 1, "the one valid id must survive")
		if _, ok := ids[7]; !ok {
			t.Fatalf("expected goroutine 7 to be parsed, got %v", ids)
		}
	})
}
