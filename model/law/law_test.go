// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

type kv struct{ data map[string]string }

func newKV() *kv { return &kv{data: make(map[string]string)} }

var errMissing = errors.New("missing")

func TestReadAfterWriteCheck(t *testing.T) {
	t.Parallel()

	t.Run("passes when SUT and ref agree", func(t *testing.T) {
		t.Parallel()
		l := law.ReadAfterWrite[*kv, string, string]{
			Read: func(_ *rapid.T, s *kv, k string) (string, error) {
				v, ok := s.data[k]
				if !ok {
					return "", errMissing
				}
				return v, nil
			},
			Keys: rapid.Just("a"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut := newKV()
			ref := newKV()
			sut.data["a"] = "x"
			ref.data["a"] = "x"
			if err := l.Check(rt, sut, ref); err != nil {
				rt.Fatalf("unexpected error: %v", err)
			}
		})
	})

	t.Run("catches value mismatch", func(t *testing.T) {
		t.Parallel()
		l := law.ReadAfterWrite[*kv, string, string]{
			Read: func(_ *rapid.T, s *kv, k string) (string, error) {
				v, ok := s.data[k]
				if !ok {
					return "", errMissing
				}
				return v, nil
			},
			Keys: rapid.Just("a"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut := newKV()
			ref := newKV()
			sut.data["a"] = "WRONG"
			ref.data["a"] = "correct"
			if err := l.Check(rt, sut, ref); err == nil {
				rt.Fatal("should have caught value mismatch")
			}
		})
	})

	t.Run("catches error mismatch", func(t *testing.T) {
		t.Parallel()
		l := law.ReadAfterWrite[*kv, string, string]{
			Read: func(_ *rapid.T, s *kv, k string) (string, error) {
				v, ok := s.data[k]
				if !ok {
					return "", errMissing
				}
				return v, nil
			},
			Keys: rapid.Just("a"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut := newKV()
			ref := newKV()
			// SUT has key, ref doesn't → error mismatch.
			sut.data["a"] = "v"
			if err := l.Check(rt, sut, ref); err == nil {
				rt.Fatal("should have caught error mismatch")
			}
		})
	})

	t.Run("passes when both return error", func(t *testing.T) {
		t.Parallel()
		l := law.ReadAfterWrite[*kv, string, string]{
			Read: func(_ *rapid.T, s *kv, k string) (string, error) {
				v, ok := s.data[k]
				if !ok {
					return "", errMissing
				}
				return v, nil
			},
			Keys: rapid.Just("missing"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut := newKV()
			ref := newKV()
			if err := l.Check(rt, sut, ref); err != nil {
				rt.Fatalf("both missing should agree: %v", err)
			}
		})
	})
}

func TestDeleteReturnsNotFoundCheck(t *testing.T) {
	t.Parallel()

	t.Run("passes when both return sentinel", func(t *testing.T) {
		t.Parallel()
		l := law.DeleteReturnsNotFound[*kv, string, string]{
			Read: func(_ *rapid.T, s *kv, k string) (string, error) {
				if _, ok := s.data[k]; !ok {
					return "", errMissing
				}
				return s.data[k], nil
			},
			Keys:     rapid.Just("a"),
			Sentinel: errMissing,
		}
		rapid.Check(t, func(rt *rapid.T) {
			// Both empty → both return sentinel → pass.
			if err := l.Check(rt, newKV(), newKV()); err != nil {
				rt.Fatalf("unexpected: %v", err)
			}
		})
	})

	t.Run("skips when ref has the key", func(t *testing.T) {
		t.Parallel()
		l := law.DeleteReturnsNotFound[*kv, string, string]{
			Read: func(_ *rapid.T, s *kv, k string) (string, error) {
				if _, ok := s.data[k]; !ok {
					return "", errMissing
				}
				return s.data[k], nil
			},
			Keys:     rapid.Just("a"),
			Sentinel: errMissing,
		}
		rapid.Check(t, func(rt *rapid.T) {
			ref := newKV()
			ref.data["a"] = "v"
			if err := l.Check(rt, newKV(), ref); err != nil {
				rt.Fatalf("should skip when ref has key: %v", err)
			}
		})
	})

	t.Run("catches SUT not returning sentinel", func(t *testing.T) {
		t.Parallel()
		l := law.DeleteReturnsNotFound[*kv, string, string]{
			Read: func(_ *rapid.T, s *kv, k string) (string, error) {
				if _, ok := s.data[k]; !ok {
					return "", errMissing
				}
				return s.data[k], nil
			},
			Keys:     rapid.Just("a"),
			Sentinel: errMissing,
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut := newKV()
			sut.data["a"] = "SHOULD-BE-DELETED"
			ref := newKV() // ref says missing
			if err := l.Check(rt, sut, ref); err == nil {
				rt.Fatal("should catch SUT not returning sentinel")
			}
		})
	})
}

func TestCountEqualsReferenceCheck(t *testing.T) {
	t.Parallel()

	t.Run("passes when counts match", func(t *testing.T) {
		t.Parallel()
		l := law.CountEqualsReference[*kv, int]{
			Count: func(_ *rapid.T, s *kv) (int, error) { return len(s.data), nil },
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut := newKV()
			ref := newKV()
			sut.data["a"] = "v"
			ref.data["a"] = "v"
			if err := l.Check(rt, sut, ref); err != nil {
				rt.Fatalf("unexpected: %v", err)
			}
		})
	})

	t.Run("catches count mismatch", func(t *testing.T) {
		t.Parallel()
		l := law.CountEqualsReference[*kv, int]{
			Count: func(_ *rapid.T, s *kv) (int, error) { return len(s.data), nil },
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut := newKV()
			ref := newKV()
			ref.data["a"] = "v" // ref has 1, SUT has 0
			if err := l.Check(rt, sut, ref); err == nil {
				rt.Fatal("should catch count mismatch")
			}
		})
	})

	t.Run("catches SUT count error", func(t *testing.T) {
		t.Parallel()
		countErr := errors.New("count failed")
		l := law.CountEqualsReference[*kv, int]{
			Count: func(_ *rapid.T, s *kv) (int, error) {
				if len(s.data) == 0 {
					return 0, countErr
				}
				return len(s.data), nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut := newKV() // empty → error
			ref := newKV()
			if err := l.Check(rt, sut, ref); err == nil {
				rt.Fatal("should catch count error")
			}
		})
	})
}

// uniqueStringGen returns a generator producing a distinct string on
// every draw. Negative tests for convergence laws need this: with
// sampled values, rapid can route the same value everywhere and a
// broken merge/sync converges by accident.
func uniqueStringGen() *rapid.Generator[string] {
	seq := 0
	return rapid.Custom(func(rt *rapid.T) string {
		// rapid requires Custom generators to consume bitstream
		// data; the value itself comes from the counter.
		rapid.Bool().Draw(rt, "pad")
		seq++
		return strconv.Itoa(seq)
	})
}

// gset is a grow-only-set CRDT: merge is set union, so replicas
// converge regardless of merge direction. When lossy is set, merge
// ignores the source — the bug where replicas never converge.
type gset struct {
	elems map[string]struct{}
	lossy bool
}

func newGSet(lossy bool) *gset {
	return &gset{elems: map[string]struct{}{}, lossy: lossy}
}

func (s *gset) add(v string) error {
	s.elems[v] = struct{}{}
	return nil
}

func (s *gset) merge(src *gset) error {
	if s.lossy {
		return nil // BUG: discards the source replica's elements
	}
	for v := range src.elems {
		s.elems[v] = struct{}{}
	}
	return nil
}

func (s *gset) sorted() []string {
	out := make([]string, 0, len(s.elems))
	for v := range s.elems {
		out = append(out, v)
	}
	slices.Sort(out)
	return out
}

func TestCRDTMerge(t *testing.T) {
	t.Parallel()

	t.Run("g-set replicas converge under bidirectional merge", func(t *testing.T) {
		t.Parallel()
		l := law.CRDTMerge[*gset, string, []string]{
			Factory: func() *gset { return newGSet(false) },
			Write:   func(_ *rapid.T, s *gset, v string) error { return s.add(v) },
			Merge:   func(_ *rapid.T, dst, src *gset) error { return dst.merge(src) },
			Values:  rapid.SampledFrom([]string{"a", "b", "c", "d"}),
			Observe: func(_ *rapid.T, s *gset) []string { return s.sorted() },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, newGSet(false), newGSet(false)); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("merge that drops the source replica is caught", func(t *testing.T) {
		t.Parallel()
		// Distinct values per draw: with a lossy (no-op) merge, any
		// split of unique values leaves the replicas divergent, so
		// the law must fire on every iteration. Sampled values could
		// route the same value to both replicas and converge by
		// accident.
		l := law.CRDTMerge[*gset, string, []string]{
			Factory: func() *gset { return newGSet(true) },
			Write:   func(_ *rapid.T, s *gset, v string) error { return s.add(v) },
			Merge:   func(_ *rapid.T, dst, src *gset) error { return dst.merge(src) },
			Values:  uniqueStringGen(),
			Observe: func(_ *rapid.T, s *gset) []string { return s.sorted() },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, newGSet(true), newGSet(true)); err == nil {
				rt.Fatal("expected non-converging merge to be caught")
			}
		})
	})
}
