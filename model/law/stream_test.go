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

func TestStreamCompletion(t *testing.T) {
	t.Parallel()

	t.Run("drain under limit passes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamCompletion[[]string, string]{
			Drain: func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Limit: 100,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "b"}, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("drain at limit flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamCompletion[[]string, string]{
			Drain: func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Limit: 3,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "b", "c", "d"}, nil); err == nil {
				rt.Fatal("expected limit reached")
			}
		})
	})
}

func TestStreamNoDuplicates(t *testing.T) {
	t.Parallel()

	t.Run("unique drain passes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamNoDuplicates[[]string, string, string]{
			Drain: func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Hash:  func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "b", "c"}, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("duplicate drain flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamNoDuplicates[[]string, string, string]{
			Drain: func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Hash:  func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "b", "a"}, nil); err == nil {
				rt.Fatal("expected duplicate")
			}
		})
	})
}

func TestStreamStableOrder(t *testing.T) {
	t.Parallel()

	t.Run("sorted drain passes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamStableOrder[[]int, int]{
			Drain: func(_ *rapid.T, s []int) ([]int, error) { return s, nil },
			Less:  func(a, b int) bool { return a < b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []int{1, 2, 3}, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("out-of-order drain flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamStableOrder[[]int, int]{
			Drain: func(_ *rapid.T, s []int) ([]int, error) { return s, nil },
			Less:  func(a, b int) bool { return a < b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []int{3, 1, 2}, nil); err == nil {
				rt.Fatal("expected out-of-order")
			}
		})
	})
}

func TestStreamPermutation(t *testing.T) {
	t.Parallel()

	t.Run("drain that permutes expected passes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamPermutation[[]string, string, string]{
			Drain:    func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Expected: func(_ *rapid.T, _ []string) []string { return []string{"a", "b", "c"} },
			Hash:     func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"c", "a", "b"}, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("length mismatch flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamPermutation[[]string, string, string]{
			Drain:    func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Expected: func(_ *rapid.T, _ []string) []string { return []string{"a", "b"} },
			Hash:     func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a"}, nil); err == nil {
				rt.Fatal("expected length mismatch")
			}
		})
	})

	t.Run("element mismatch flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamPermutation[[]string, string, string]{
			Drain:    func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Expected: func(_ *rapid.T, _ []string) []string { return []string{"a", "b"} },
			Hash:     func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "x"}, nil); err == nil {
				rt.Fatal("expected element mismatch")
			}
		})
	})
}

func TestStreamOverMatch(t *testing.T) {
	t.Parallel()

	t.Run("drain containing required passes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamOverMatch[[]string, string, string]{
			Drain:    func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Required: func(_ *rapid.T, _ []string) []string { return []string{"a", "b"} },
			Hash:     func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "b", "c", "d"}, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("missing required element flagged", func(t *testing.T) {
		t.Parallel()
		l := law.StreamOverMatch[[]string, string, string]{
			Drain:    func(_ *rapid.T, s []string) ([]string, error) { return s, nil },
			Required: func(_ *rapid.T, _ []string) []string { return []string{"a", "b"} },
			Hash:     func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, []string{"a", "c"}, nil); err == nil {
				rt.Fatal("expected missing element")
			}
		})
	})

	t.Run("drain error is vacuous", func(t *testing.T) {
		t.Parallel()
		l := law.StreamOverMatch[[]string, string, string]{
			Drain:    func(_ *rapid.T, _ []string) ([]string, error) { return nil, errors.New("nope") },
			Required: func(_ *rapid.T, _ []string) []string { return []string{"a"} },
			Hash:     func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

// snapStore is a keyed store whose stream either reflects live state
// or (when stale is set) a snapshot frozen at construction — the bug
// where mutations never show up in the streamed view.
type snapStore struct {
	live  map[string]string
	snap  map[string]string
	stale bool
}

func newSnapStore(stale bool) *snapStore {
	return &snapStore{live: map[string]string{}, snap: map[string]string{}, stale: stale}
}

func (s *snapStore) put(v string) error {
	s.live[v] = v
	return nil
}

func (s *snapStore) del(v string) error {
	delete(s.live, v)
	return nil
}

func (s *snapStore) stream() ([]string, error) {
	src := s.live
	if s.stale {
		src = s.snap
	}
	out := make([]string, 0, len(src))
	for v := range src {
		out = append(out, v)
	}
	return out, nil
}

func TestStreamReflectsMutations(t *testing.T) {
	t.Parallel()

	t.Run("live stream reflects puts and deletes", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReflectsMutations[*snapStore, string, string]{
			Put:    func(_ *rapid.T, s *snapStore, v string) error { return s.put(v) },
			Delete: func(_ *rapid.T, s *snapStore, v string) error { return s.del(v) },
			Drain:  func(_ *rapid.T, s *snapStore) ([]string, error) { return s.stream() },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			Hash:   func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newSnapStore(false)
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("stale-snapshot stream is caught", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReflectsMutations[*snapStore, string, string]{
			Put:    func(_ *rapid.T, s *snapStore, v string) error { return s.put(v) },
			Delete: func(_ *rapid.T, s *snapStore, v string) error { return s.del(v) },
			Drain:  func(_ *rapid.T, s *snapStore) ([]string, error) { return s.stream() },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			Hash:   func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newSnapStore(true)
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected stale stream to be caught")
			}
		})
	})

	t.Run("delete omitted checks only the put direction", func(t *testing.T) {
		t.Parallel()
		l := law.StreamReflectsMutations[*snapStore, string, string]{
			Put:    func(_ *rapid.T, s *snapStore, v string) error { return s.put(v) },
			Drain:  func(_ *rapid.T, s *snapStore) ([]string, error) { return s.stream() },
			Values: rapid.SampledFrom([]string{"a", "b"}),
			Hash:   func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newSnapStore(false)
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})
}
