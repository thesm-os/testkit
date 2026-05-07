// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"sync/atomic"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

func TestStreamReentrancy(t *testing.T) {
	t.Parallel()

	t.Run("passes for reentrant stream", func(t *testing.T) {
		t.Parallel()
		items := []string{"a", "b", "c"}
		l := law.StreamReentrancy[[]string, string]{
			Collect: func(_ *rapid.T, s []string) ([]string, error) {
				return s, nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, items, nil)
			if err != nil {
				rt.Fatalf("unexpected error: %v", err)
			}
		})
	})

	t.Run("detects one-shot iterator", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReentrancy[*atomic.Int64, string]{
			Collect: func(_ *rapid.T, counter *atomic.Int64) ([]string, error) {
				n := counter.Add(1)
				if n > 1 {
					// BUG: second iteration returns empty.
					return nil, nil
				}
				return []string{"item"}, nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			counter := &atomic.Int64{}
			err := l.Check(rt, counter, nil)
			if err == nil {
				rt.Fatal("should have detected one-shot iterator")
			}
		})
	})

	t.Run("passes for empty stream", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReentrancy[string, int]{
			Collect: func(_ *rapid.T, _ string) ([]int, error) {
				return nil, nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, "x", "x")
			if err != nil {
				rt.Fatalf("unexpected error: %v", err)
			}
		})
	})

	t.Run("detects error on second iteration", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReentrancy[*atomic.Int64, string]{
			Collect: func(_ *rapid.T, counter *atomic.Int64) ([]string, error) {
				n := counter.Add(1)
				if n > 1 {
					return nil, errors.New("second iteration fails")
				}
				return []string{"item"}, nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			counter := &atomic.Int64{}
			err := l.Check(rt, counter, nil)
			if err == nil {
				rt.Fatal("should have detected error on second iteration")
			}
		})
	})
}
