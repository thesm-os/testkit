// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"sync"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
)

func TestRecorder(t *testing.T) {
	t.Parallel()

	t.Run("Record and Calls", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[string]()
		rec.Record("a")
		rec.Record("b")
		calls := rec.Calls()
		testkit.Equal(t, calls, []string{"a", "b"}, "must capture calls in order")
	})

	t.Run("Calls returns defensive copy", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[string]()
		rec.Record("a")
		calls := rec.Calls()
		calls[0] = "mutated"
		testkit.Equal(t, rec.Calls()[0], "a", "must not be aliased")
	})

	t.Run("CallCount", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		testkit.Equal(t, rec.CallCount(), 0, "new recorder has zero calls")
		rec.Record(1)
		rec.Record(2)
		testkit.Equal(t, rec.CallCount(), 2, "must count recorded calls")
	})

	t.Run("LastCall returns most recent", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[string]()
		rec.Record("first")
		rec.Record("last")
		testkit.Equal(t, rec.LastCall(t), "last", "must return last recorded value")
	})

	t.Run("LastCall fatals on empty recorder", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		rec := testkit.NewRecorder[string]()
		rec.LastCall(f)
		testkit.True(t, f.Failed(), "must fatal on empty recorder")
	})

	t.Run("Reset clears calls", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		rec.Record(1)
		rec.Reset()
		testkit.Equal(t, rec.CallCount(), 0, "must be empty after reset")
	})
}

func TestRecorder_assertions(t *testing.T) {
	t.Parallel()

	t.Run("AssertCalledOnce passes with one call", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[string]()
		rec.Record("only")
		v := rec.AssertCalledOnce(t, "single call")
		testkit.Equal(t, v, "only", "must return the call")
	})

	t.Run("AssertCalledOnce fails with zero calls", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		rec := testkit.NewRecorder[string]()
		rec.AssertCalledOnce(f, "single call")
		testkit.True(t, f.Failed(), "must fail with zero calls")
	})

	t.Run("AssertCalledOnce fails with multiple calls", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		rec := testkit.NewRecorder[string]()
		rec.Record("a")
		rec.Record("b")
		rec.AssertCalledOnce(f, "single call")
		testkit.True(t, f.Failed(), "must fail with multiple calls")
	})

	t.Run("AssertCalledN passes with exact count", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		rec.Record(1)
		rec.Record(2)
		calls := rec.AssertCalledN(t, 2, "two calls")
		testkit.Equal(t, calls, []int{1, 2}, "must return all calls")
	})

	t.Run("AssertCalledN fails with wrong count", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		rec := testkit.NewRecorder[int]()
		rec.Record(1)
		rec.AssertCalledN(f, 5, "five calls")
		testkit.True(t, f.Failed(), "must fail with wrong count")
	})

	t.Run("AssertNotCalled passes with no calls", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[string]()
		rec.AssertNotCalled(t, "no calls")
	})

	t.Run("AssertNotCalled fails with calls", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		rec := testkit.NewRecorder[string]()
		rec.Record("a")
		rec.AssertNotCalled(f, "no calls")
		testkit.True(t, f.Failed(), "must fail when calls exist")
	})
}

func TestRecorder_filtering(t *testing.T) {
	t.Parallel()

	t.Run("Filter returns matching calls", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		rec.Record(1)
		rec.Record(2)
		rec.Record(3)
		evens := rec.Filter(func(v int) bool { return v%2 == 0 })
		testkit.Equal(t, evens, []int{2}, "must return only evens")
	})

	t.Run("First returns first match", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		rec.Record(1)
		rec.Record(2)
		rec.Record(3)
		v, ok := rec.First(func(v int) bool { return v > 1 })
		testkit.True(t, ok, "must find a match")
		testkit.Equal(t, v, 2, "must return first match")
	})

	t.Run("First returns false when no match", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		rec.Record(1)
		_, ok := rec.First(func(v int) bool { return v > 100 })
		testkit.False(t, ok, "must not find a match")
	})

	t.Run("Any reports true when match exists", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		rec.Record(1)
		rec.Record(2)
		testkit.True(t, rec.Any(func(v int) bool { return v == 2 }), "must find 2")
	})

	t.Run("Any reports false when no match", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		rec.Record(1)
		testkit.False(t, rec.Any(func(v int) bool { return v == 99 }), "must not find 99")
	})

	t.Run("All returns true when all match", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		rec.Record(2)
		rec.Record(4)
		testkit.True(t, rec.All(func(v int) bool { return v%2 == 0 }), "all must be even")
	})

	t.Run("All returns false when one fails", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		rec.Record(2)
		rec.Record(3)
		testkit.False(t, rec.All(func(v int) bool { return v%2 == 0 }), "3 is not even")
	})

	t.Run("All returns true for empty recorder", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		testkit.True(t, rec.All(func(v int) bool { return false }), "empty must be vacuously true")
	})
}

func TestRecorder_waiting(t *testing.T) {
	t.Parallel()

	t.Run("WaitForN unblocks when count reached", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		go func() {
			time.Sleep(10 * time.Millisecond) //nolint:testifylint // deliberate async delay
			rec.Record(1)
			rec.Record(2)
		}()
		rec.WaitForN(t, 2, time.Second)
		testkit.Equal(t, rec.CallCount(), 2, "must have 2 calls after wait")
	})

	t.Run("WaitForN fatals on timeout", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		rec := testkit.NewRecorder[int]()
		rec.WaitForN(f, 5, 50*time.Millisecond)
		testkit.True(t, f.Failed(), "must fatal on timeout")
	})

	t.Run("WaitFor unblocks on matching call", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[string]()
		go func() {
			time.Sleep(10 * time.Millisecond) //nolint:testifylint // deliberate async delay
			rec.Record("not-this")
			rec.Record("target")
		}()
		rec.WaitFor(t, func(v string) bool { return v == "target" }, time.Second, "must find target")
	})

	t.Run("WaitFor fatals on timeout", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		rec := testkit.NewRecorder[string]()
		rec.WaitFor(f, func(v string) bool { return v == "never" }, 50*time.Millisecond, "target")
		testkit.True(t, f.Failed(), "must fatal on timeout")
	})
}

func TestRecorder_hooks(t *testing.T) {
	t.Parallel()

	t.Run("OnRecord fires on each Record call", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		var seen []int
		rec.OnRecord(func(v int) {
			seen = append(seen, v)
		})
		rec.Record(1)
		rec.Record(2)
		testkit.Equal(t, seen, []int{1, 2}, "hook must see all values")
	})

	t.Run("multiple hooks fire in order", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[string]()
		var order []string
		rec.OnRecord(func(_ string) { order = append(order, "hook-a") })
		rec.OnRecord(func(_ string) { order = append(order, "hook-b") })
		rec.Record("x")
		testkit.Equal(t, order, []string{"hook-a", "hook-b"}, "hooks must fire in registration order")
	})
}

func TestRecorder_gating(t *testing.T) {
	t.Parallel()

	t.Run("gate blocks Record until Release", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[string]()
		gate := rec.NewGate()

		var wg sync.WaitGroup
		wg.Go(func() {
			rec.Record("blocked")
		})

		// Give goroutine time to block.
		time.Sleep(20 * time.Millisecond) //nolint:testifylint // deliberate race setup
		testkit.Equal(t, rec.CallCount(), 0, "must be blocked before release")

		gate.Release()
		wg.Wait()
		testkit.Equal(t, rec.CallCount(), 1, "must unblock after release")
	})

	t.Run("ReleaseOne unblocks exactly one", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		gate := rec.NewGate()

		var wg sync.WaitGroup
		wg.Go(func() { rec.Record(1) })
		wg.Go(func() { rec.Record(2) })

		time.Sleep(20 * time.Millisecond) //nolint:testifylint // deliberate race setup
		testkit.Equal(t, rec.CallCount(), 0, "both must be blocked")

		gate.ReleaseOne()
		time.Sleep(20 * time.Millisecond) //nolint:testifylint // let one unblock
		testkit.Equal(t, rec.CallCount(), 1, "exactly one must unblock")

		gate.ReleaseOne()
		wg.Wait()
		testkit.Equal(t, rec.CallCount(), 2, "both must complete")
	})
}

func TestRecorder_concurrent_safety(t *testing.T) {
	t.Parallel()

	t.Run("concurrent Record calls do not race", func(t *testing.T) {
		t.Parallel()
		rec := testkit.NewRecorder[int]()
		var wg sync.WaitGroup
		for i := range 100 {
			wg.Go(func() {
				rec.Record(i)
			})
		}
		wg.Wait()
		testkit.Equal(t, rec.CallCount(), 100, "must record all concurrent calls")
	})
}
