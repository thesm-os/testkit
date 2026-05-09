// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
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
