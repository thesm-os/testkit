// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"sync/atomic"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
)

func TestPureDeterminism(t *testing.T) {
	t.Parallel()

	t.Run("passes for deterministic function", func(t *testing.T) {
		t.Parallel()
		l := law.PureDeterminism[string, int]{
			Call: func(_ *rapid.T, s string) int { return len(s) },
		}
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, "hello", "hello")
			if err != nil {
				rt.Fatalf("unexpected error: %v", err)
			}
		})
	})

	t.Run("detects nondeterministic function", func(t *testing.T) {
		t.Parallel()
		var counter atomic.Int64
		l := law.PureDeterminism[string, int64]{
			Call: func(_ *rapid.T, _ string) int64 {
				return counter.Add(1)
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, "x", "x")
			if err == nil {
				rt.Fatal("should have detected nondeterminism")
			}
		})
	})

	t.Run("defaults N to 3 when zero", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int64
		l := law.PureDeterminism[string, string]{
			Call: func(_ *rapid.T, s string) string {
				calls.Add(1)
				return s
			},
			N: 0,
		}
		rapid.Check(t, func(rt *rapid.T) {
			calls.Store(0)
			err := l.Check(rt, "test", "test")
			if err != nil {
				rt.Fatalf("unexpected error: %v", err)
			}
			if calls.Load() != 3 {
				rt.Fatalf("expected 3 calls, got %d", calls.Load())
			}
		})
	})

	t.Run("N of 1 always passes", func(t *testing.T) {
		t.Parallel()
		var counter atomic.Int64
		l := law.PureDeterminism[string, int64]{
			Call: func(_ *rapid.T, _ string) int64 {
				return counter.Add(1)
			},
			N: 1,
		}
		rapid.Check(t, func(rt *rapid.T) {
			// N=1 means only one call — no comparison, always passes.
			err := l.Check(rt, "x", "x")
			if err != nil {
				rt.Fatalf("N=1 should always pass: %v", err)
			}
		})
	})
}
