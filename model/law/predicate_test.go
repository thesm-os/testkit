// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"sync/atomic"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

func TestPredicateConsistency(t *testing.T) {
	t.Parallel()

	t.Run("passes for stable predicate", func(t *testing.T) {
		t.Parallel()
		l := law.PredicateConsistency[string]{
			Call: func(_ *rapid.T, s string) bool { return len(s) > 0 },
		}
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, "hello", "hello")
			if err != nil {
				rt.Fatalf("unexpected error: %v", err)
			}
		})
	})

	t.Run("detects flapping predicate", func(t *testing.T) {
		t.Parallel()
		var toggle atomic.Int64
		l := law.PredicateConsistency[string]{
			Call: func(_ *rapid.T, _ string) bool {
				return toggle.Add(1)%2 == 0
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, "x", "x")
			if err == nil {
				rt.Fatal("should have detected flapping predicate")
			}
		})
	})

	t.Run("defaults N to 3 when zero", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int64
		l := law.PredicateConsistency[string]{
			Call: func(_ *rapid.T, _ string) bool {
				calls.Add(1)
				return true
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
}
