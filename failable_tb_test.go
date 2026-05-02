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
		f.Fatal("first")
		f.Fatal("second")
		if f.Msg() != "first" {
			t.Fatalf("should capture first message, got: %q", f.Msg())
		}
	})

	t.Run("Errorf overwrites message each time", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Errorf("first")
		f.Errorf("second")
		if f.Msg() != "second" {
			t.Fatalf("Errorf should overwrite, got: %q", f.Msg())
		}
		if !f.Failed() {
			t.Fatal("should be failed after Errorf")
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

	t.Run("Cleanup does not panic", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		f.Cleanup(func() {}) // no-op, should not panic
	})

	t.Run("TempDir returns empty string", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		if f.TempDir() != "" {
			t.Fatal("TempDir should return empty string")
		}
	})
}
