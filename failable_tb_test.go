// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
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
