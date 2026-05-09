// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/suite"
)

func TestHookRecorder(t *testing.T) {
	t.Parallel()

	t.Run("NewHookRecorder yields a recorder with zero counts", func(t *testing.T) {
		t.Parallel()
		r := suite.NewHookRecorder()
		testkit.Equal(t, r.Count("anything"), 0, "fresh recorder has zero counts")
	})

	t.Run("Record increments count for named hook", func(t *testing.T) {
		t.Parallel()
		r := suite.NewHookRecorder()
		r.Record("BeforeWrite")
		r.Record("BeforeWrite")
		r.Record("AfterWrite")
		testkit.Equal(t, r.Count("BeforeWrite"), 2, "BeforeWrite recorded twice")
		testkit.Equal(t, r.Count("AfterWrite"), 1, "AfterWrite recorded once")
		testkit.Equal(t, r.Count("Other"), 0, "untouched hook has zero count")
	})

	t.Run("Record on nil recorder is a no-op (production path)", func(t *testing.T) {
		t.Parallel()
		var r *suite.HookRecorder
		// no panic
		r.Record("X")
		testkit.Equal(t, r.Count("X"), 0, "nil recorder always reports zero")
	})

	t.Run("ContextWithRecorder + RecorderFromContext round-trip", func(t *testing.T) {
		t.Parallel()
		r := suite.NewHookRecorder()
		ctx := suite.ContextWithRecorder(t.Context(), r)
		got := suite.RecorderFromContext(ctx)
		testkit.True(t, got == r, "recorder retrieved equals registered")
	})

	t.Run("RecorderFromContext returns nil when absent", func(t *testing.T) {
		t.Parallel()
		got := suite.RecorderFromContext(t.Context())
		testkit.True(t, got == nil, "no recorder => nil")
	})

	t.Run("Record is goroutine-safe", func(t *testing.T) {
		t.Parallel()
		r := suite.NewHookRecorder()
		var wg sync.WaitGroup
		for range 16 {
			wg.Go(func() {
				for range 100 {
					r.Record("Concurrent")
				}
			})
		}
		wg.Wait()
		testkit.Equal(t, r.Count("Concurrent"), 16*100, "concurrent records sum correctly")
	})

	t.Run("RecorderFromContext ignores wrong-typed values", func(t *testing.T) {
		t.Parallel()
		// Construct a context with a non-HookRecorder value under
		// some unrelated key — should not affect the helper.
		type otherKey struct{}
		ctx := context.WithValue(t.Context(), otherKey{}, "irrelevant")
		got := suite.RecorderFromContext(ctx)
		testkit.True(t, got == nil, "unrelated context value yields nil recorder")
	})
}
