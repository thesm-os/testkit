// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
)

type cacheableSUT struct {
	values map[string]int
}

func TestCacheable(t *testing.T) {
	t.Parallel()

	t.Run("repeated read returns same value", func(t *testing.T) {
		t.Parallel()
		s := &cacheableSUT{values: map[string]int{"k": 42}}
		l := law.Cacheable[*cacheableSUT, string, int]{
			Keys: rapid.Just("k"),
			Read: func(_ *rapid.T, c *cacheableSUT, k string) (int, error) {
				return c.values[k], nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("flips between calls catches mismatch", func(t *testing.T) {
		t.Parallel()
		toggle := false
		l := law.Cacheable[*cacheableSUT, string, int]{
			Keys: rapid.Just("k"),
			Read: func(_ *rapid.T, _ *cacheableSUT, _ string) (int, error) {
				toggle = !toggle
				if toggle {
					return 1, nil
				}
				return 2, nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &cacheableSUT{}, &cacheableSUT{}); err == nil {
				rt.Fatal("expected mismatch")
			}
		})
	})
}

func TestDefaultOnError(t *testing.T) {
	t.Parallel()

	t.Run("error-coupled default passes", func(t *testing.T) {
		t.Parallel()
		l := law.DefaultOnError[*cacheableSUT, string, int]{
			Keys: rapid.Just("k"),
			Read: func(_ *rapid.T, _ *cacheableSUT, _ string) (int, error) {
				return 0, errors.New("boom")
			},
			Default: 0,
			Eq:      func(a, b int) bool { return a == b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &cacheableSUT{}, &cacheableSUT{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("non-default-on-error flagged", func(t *testing.T) {
		t.Parallel()
		l := law.DefaultOnError[*cacheableSUT, string, int]{
			Keys: rapid.Just("k"),
			Read: func(_ *rapid.T, _ *cacheableSUT, _ string) (int, error) {
				return 99, errors.New("boom")
			},
			Default: 0,
			Eq:      func(a, b int) bool { return a == b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &cacheableSUT{}, &cacheableSUT{}); err == nil {
				rt.Fatal("expected mismatch")
			}
		})
	})
}

func TestSticky(t *testing.T) {
	t.Parallel()

	t.Run("first observation persists across calls", func(t *testing.T) {
		t.Parallel()
		l := &law.Sticky[*cacheableSUT, string, int]{
			Keys: rapid.Just("k"),
			Read: func(_ *rapid.T, _ *cacheableSUT, _ string) (int, error) {
				return 42, nil
			},
			Eq: func(a, b int) bool { return a == b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &cacheableSUT{}, &cacheableSUT{}); err != nil {
				rt.Fatal(err)
			}
		})
	})
}
