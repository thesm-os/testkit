// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
)

func TestFailableTB(t *testing.T) {
	t.Parallel()

	t.Run("new instance is not failed", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		if f.Failed() {
			t.Fatal("new FailableTB should not be failed")
		}
		if f.Msg() != "" {
			t.Fatal("new FailableTB should have empty message")
		}
	})

	t.Run("Fatalf records first failure only", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Fatalf("first: %d", 1)
		f.Fatalf("second: %d", 2)
		if !f.Failed() {
			t.Fatal("should be failed after Fatalf")
		}
		if f.Msg() != "first: 1" {
			t.Fatalf("should capture first message, got: %q", f.Msg())
		}
	})

	t.Run("Fatal records first failure only", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Fatal("fatal-one")
		f.Fatal("fatal-two")
		if f.Msg() != "fatal-one" {
			t.Fatalf("should capture first message, got: %q", f.Msg())
		}
	})

	t.Run("FailNow marks failed", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.FailNow()
		if !f.Failed() {
			t.Fatal("should be failed after FailNow")
		}
	})

	t.Run("Fail marks failed without terminating", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Fail()
		if !f.Failed() {
			t.Fatal("should be failed after Fail")
		}
	})

	t.Run("Errorf overwrites message each time", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Errorf("err-one")
		f.Errorf("err-two")
		if f.Msg() != "err-two" {
			t.Fatalf("Errorf should overwrite, got: %q", f.Msg())
		}
		if !f.Failed() {
			t.Fatal("should be failed after Errorf")
		}
	})

	t.Run("Error marks failed", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Error("oops")
		if !f.Failed() {
			t.Fatal("should be failed after Error")
		}
		if f.Msg() != "oops" {
			t.Fatalf("expected oops, got: %q", f.Msg())
		}
	})

	t.Run("Logf appends messages", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Logf("a: %d", 1)
		f.Logf("b: %d", 2)
		logs := f.Logs()
		if len(logs) != 2 {
			t.Fatalf("expected 2 logs, got %d", len(logs))
		}
		if logs[0] != "a: 1" || logs[1] != "b: 2" {
			t.Fatalf("unexpected logs: %v", logs)
		}
	})

	t.Run("Log appends messages", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Log("hello")
		logs := f.Logs()
		if len(logs) != 1 || logs[0] != "hello" {
			t.Fatalf("expected [hello], got: %v", logs)
		}
	})

	t.Run("Logs returns defensive copy", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Logf("original")
		logs := f.Logs()
		logs[0] = "mutated"
		if f.Logs()[0] != "original" {
			t.Fatal("Logs should return a defensive copy")
		}
	})

	t.Run("Name defaults to FailableTB", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		if f.Name() != "FailableTB" {
			t.Fatalf("expected FailableTB, got: %q", f.Name())
		}
	})

	t.Run("WithName sets custom name", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB().WithName("custom")
		if f.Name() != "custom" {
			t.Fatalf("expected custom, got: %q", f.Name())
		}
	})

	t.Run("Helper increments counter", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Helper()
		f.Helper()
		if f.HelperCalls() != 2 {
			t.Fatalf("expected 2 helper calls, got %d", f.HelperCalls())
		}
	})

	t.Run("Context returns non-nil context", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		if f.Context() == nil {
			t.Fatal("Context should return non-nil")
		}
	})

	t.Run("Context is cancelled after Fatal", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		ctx := f.Context()
		if ctx.Err() != nil {
			t.Fatal("context should not be cancelled before Fatal")
		}
		f.Fatal("boom")
		if ctx.Err() == nil {
			t.Fatal("context should be cancelled after Fatal")
		}
	})

	t.Run("Context is cancelled after Fatalf", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		ctx := f.Context()
		f.Fatalf("boom: %d", 1)
		if ctx.Err() == nil {
			t.Fatal("context should be cancelled after Fatalf")
		}
	})

	t.Run("Errorf does not cancel context", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		ctx := f.Context()
		f.Errorf("non-fatal")
		if ctx.Err() != nil {
			t.Fatal("context should not be cancelled after Errorf")
		}
	})

	t.Run("WithGoexit terminates goroutine on Fatal", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			f.Fatal("boom")
			// This line should NOT execute — Goexit terminates the goroutine.
			f.Logf("unreachable")
		}()
		<-done
		if !f.Failed() {
			t.Fatal("should be failed after Fatal with Goexit")
		}
		if len(f.Logs()) != 0 {
			t.Fatal("unreachable code ran after Fatal with Goexit")
		}
	})

	t.Run("WithGoexit terminates goroutine on Fatalf", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			f.Fatalf("boom %d", 7)
			f.Logf("unreachable")
		}()
		<-done
		if !f.Failed() {
			t.Fatal("should be failed after Fatalf with Goexit")
		}
		if got := f.Msg(); got != "boom 7" {
			t.Fatalf("the formatted message must be recorded, got: %q", got)
		}
		if len(f.Logs()) != 0 {
			t.Fatal("unreachable code ran after Fatalf with Goexit")
		}
	})

	t.Run("WithGoexit terminates goroutine on FailNow", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			f.FailNow()
			f.Logf("unreachable")
		}()
		<-done
		if !f.Failed() {
			t.Fatal("should be failed after FailNow with Goexit")
		}
		if len(f.Logs()) != 0 {
			t.Fatal("unreachable code ran after FailNow with Goexit")
		}
	})

	t.Run("Cleanup registers function", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		called := false
		f.Cleanup(func() { called = true })
		f.RunCleanups()
		if !called {
			t.Fatal("Cleanup function must be called by RunCleanups")
		}
	})

	t.Run("RunCleanups executes in LIFO order", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		var order []int
		f.Cleanup(func() { order = append(order, 1) })
		f.Cleanup(func() { order = append(order, 2) })
		f.Cleanup(func() { order = append(order, 3) })
		f.RunCleanups()
		if len(order) != 3 || order[0] != 3 || order[1] != 2 || order[2] != 1 {
			t.Fatalf("expected LIFO [3 2 1], got %v", order)
		}
	})

	t.Run("RunCleanups with no cleanups does not panic", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.RunCleanups() // should not panic
	})

	t.Run("TempDir returns empty string", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		if f.TempDir() != "" {
			t.Fatal("TempDir should return empty string")
		}
	})
}

// Skip, Skipf and SkipNow are deliberately no-ops: a FailableTB records
// assertion outcomes, and a skipped subject is neither a pass nor a failure.
// Under WithGoexit they still terminate the goroutine, matching the control
// flow testing.T gives a real skip.
func TestFailableTBSkip(t *testing.T) {
	t.Parallel()

	t.Run("Skip is a no-op without Goexit", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Skip("not applicable")
		if f.Failed() {
			t.Fatal("Skip must not mark the subject failed")
		}
	})

	t.Run("Skipf is a no-op without Goexit", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Skipf("not applicable: %d", 1)
		if f.Failed() {
			t.Fatal("Skipf must not mark the subject failed")
		}
	})

	t.Run("SkipNow is a no-op without Goexit", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.SkipNow()
		if f.Failed() {
			t.Fatal("SkipNow must not mark the subject failed")
		}
	})

	t.Run("WithGoexit terminates the goroutine on Skip", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			f.Skip("stop here")
			f.Logf("unreachable")
		}()
		<-done
		if len(f.Logs()) != 0 {
			t.Fatal("unreachable code ran after Skip with Goexit")
		}
	})

	t.Run("WithGoexit terminates the goroutine on Skipf", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			f.Skipf("stop: %d", 1)
			f.Logf("unreachable")
		}()
		<-done
		if len(f.Logs()) != 0 {
			t.Fatal("unreachable code ran after Skipf with Goexit")
		}
	})

	t.Run("WithGoexit terminates the goroutine on SkipNow", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			f.SkipNow()
			f.Logf("unreachable")
		}()
		<-done
		if len(f.Logs()) != 0 {
			t.Fatal("unreachable code ran after SkipNow with Goexit")
		}
	})
}

// A FailableTB satisfies both interfaces. The assertion is compile-time
// because that is when it matters: a method added to BenchTB upstream, or one
// renamed here, breaks the build rather than a generated benchmark somewhere
// downstream.
var (
	_ testing.TB      = (*testkit.FailableTB)(nil)
	_ testkit.BenchTB = (*testkit.FailableTB)(nil)
)

// The benchmark half of the stand-in. Contract's own checks in contract_test.go
// drive these through StartContract; what is pinned here is the behaviour those
// checks rely on and would not notice losing.
func TestFailableTBBench(t *testing.T) {
	t.Parallel()

	t.Run("runs no iterations until bounded", func(t *testing.T) {
		t.Parallel()
		// The default is what makes "End before Loop" checkable, so a default
		// that looped would silently turn that check into a different one.
		f := testkit.NewFailableTB()
		if f.Loop() {
			t.Fatal("an unbounded FailableTB must not loop")
		}
		if got := f.Iterations(); got != 0 {
			t.Fatalf("Iterations() = %d, want 0", got)
		}
	})

	t.Run("loops exactly the bounded number of times", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB().WithIterations(3)
		n := 0
		for f.Loop() {
			n++
			if n > 10 {
				t.Fatal("Loop did not terminate")
			}
		}
		if n != 3 {
			t.Fatalf("looped %d times, want 3", n)
		}
		if got := f.Iterations(); got != 3 {
			t.Fatalf("Iterations() = %d, want 3", got)
		}
	})

	t.Run("reports no allocations until asked", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		if f.AllocsReported() {
			t.Fatal("a fresh FailableTB must not report allocations")
		}
		f.ReportAllocs()
		if !f.AllocsReported() {
			t.Fatal("ReportAllocs must be recorded")
		}
	})

	t.Run("distinguishes an absent metric from a zero one", func(t *testing.T) {
		t.Parallel()
		// A latency contract reports ns/op-p99 only while tracking latency, so
		// a caller reading zero has to be able to tell which it got.
		f := testkit.NewFailableTB()
		if _, ok := f.Metric("ns/op-p99"); ok {
			t.Fatal("an unreported metric must not be present")
		}
		f.ReportMetric(0, "ns/op-p99")
		got, ok := f.Metric("ns/op-p99")
		if !ok {
			t.Fatal("a reported metric must be present")
		}
		if got != 0 {
			t.Fatalf("Metric() = %v, want 0", got)
		}
	})

	t.Run("keeps the later value for a unit reported twice", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.ReportMetric(1, "B/op")
		f.ReportMetric(2, "B/op")
		if got, _ := f.Metric("B/op"); got != 2 {
			t.Fatalf("Metric() = %v, want the later value 2", got)
		}
	})
}

func TestGoCatchesChildGoroutinePanics(t *testing.T) {
	t.Parallel()

	f := testkit.NewFailableTB()
	f.Go(func() { panic("planted: worker dies") })
	f.Go(func() {}) // a healthy sibling must not be blamed
	f.RunCleanups()
	if !f.Failed() {
		t.Fatal("a panic in a spawned goroutine must fail the TB, not the process")
	}
	if msg := f.Msg(); !strings.Contains(msg, "planted: worker dies") {
		t.Fatalf("the panic value must reach the failure message, got %q", msg)
	}
}

func TestRunCleanupsJoinsSpawnedGoroutines(t *testing.T) {
	t.Parallel()

	f := testkit.NewFailableTB()
	done := make(chan struct{})
	f.Go(func() { <-done })
	go func() { close(done) }()
	f.RunCleanups() // must not race the worker; joining is the contract
	if f.Failed() {
		t.Fatalf("a clean worker must not fail the TB: %s", f.Msg())
	}
}
