// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

type wkv struct {
	data map[string]string
}

func (s *wkv) put(_ *rapid.T, v string) error {
	if s.data == nil {
		s.data = make(map[string]string)
	}
	s.data[v] = v
	return nil
}

func (s *wkv) observe(_ *rapid.T) string {
	var b strings.Builder
	for _, v := range s.data {
		b.WriteString(v)
	}
	return b.String()
}

func TestIdempotentWrite(t *testing.T) {
	t.Parallel()

	t.Run("repeating same Write leaves Observe unchanged", func(t *testing.T) {
		t.Parallel()
		s := &wkv{}
		l := law.IdempotentWrite[*wkv, string, string]{
			Write:   func(rt *rapid.T, w *wkv, v string) error { return w.put(rt, v) },
			Values:  rapid.Just("v"),
			Observe: func(rt *rapid.T, w *wkv) string { return w.observe(rt) },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("non-idempotent second write flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		l := law.IdempotentWrite[*wkv, string, string]{
			Write: func(_ *rapid.T, w *wkv, v string) error {
				if w.data == nil {
					w.data = make(map[string]string)
				}
				i++
				w.data[v] = v + " " + string(rune('A'+(i%26)))
				return nil
			},
			Values:  rapid.Just("v"),
			Observe: func(_ *rapid.T, w *wkv) string { return w.data["v"] },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err == nil {
				rt.Fatal("expected idempotence to be flagged")
			}
		})
	})
}

func TestAtomicWrite(t *testing.T) {
	t.Parallel()

	t.Run("erroring write that mutates is flagged", func(t *testing.T) {
		t.Parallel()
		l := law.AtomicWrite[*wkv, string, string]{
			Write: func(_ *rapid.T, w *wkv, v string) error {
				if w.data == nil {
					w.data = make(map[string]string)
				}
				w.data[v] = v
				return errors.New("boom")
			},
			Values:  rapid.Just("v"),
			Observe: func(_ *rapid.T, w *wkv) string { return w.data["v"] },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err == nil {
				rt.Fatal("expected atomicity to be flagged")
			}
		})
	})

	t.Run("erroring write that does not mutate passes", func(t *testing.T) {
		t.Parallel()
		l := law.AtomicWrite[*wkv, string, string]{
			Write:   func(_ *rapid.T, _ *wkv, _ string) error { return errors.New("boom") },
			Values:  rapid.Just("v"),
			Observe: func(_ *rapid.T, w *wkv) string { return w.observe(nil) },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestCommutativeWrite(t *testing.T) {
	t.Parallel()

	t.Run("a;b == b;a for set-style writes", func(t *testing.T) {
		t.Parallel()
		l := law.CommutativeWrite[*wkv, string, string]{
			Factory: func() *wkv { return &wkv{data: make(map[string]string)} },
			Write:   func(_ *rapid.T, w *wkv, v string) error { w.data[v] = v; return nil },
			Values:  rapid.SampledFrom([]string{"a", "b", "c"}),
			Observe: func(_ *rapid.T, w *wkv) string {
				// map iteration → sort via observe()'s deterministic
				// concat path; commutative writes converge here.
				keys := make([]byte, 0, len(w.data))
				for k := range w.data {
					if len(k) > 0 {
						keys = append(keys, k[0])
					}
				}
				// Sort.
				for i := 1; i < len(keys); i++ {
					for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
						keys[j], keys[j-1] = keys[j-1], keys[j]
					}
				}
				return string(keys)
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestValidTransition(t *testing.T) {
	t.Parallel()

	t.Run("allowed transitions pass", func(t *testing.T) {
		t.Parallel()
		l := law.ValidTransition[*wkv, string, int]{
			Write:   func(_ *rapid.T, w *wkv, _ string) error { w.data = map[string]string{"x": "y"}; return nil },
			Values:  rapid.Just("v"),
			Observe: func(_ *rapid.T, w *wkv) int { return len(w.data) },
			Allowed: func(from, to int) bool { return to >= from },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("disallowed transition flagged", func(t *testing.T) {
		t.Parallel()
		l := law.ValidTransition[*wkv, string, int]{
			Write:   func(_ *rapid.T, w *wkv, _ string) error { w.data = map[string]string{"x": "y"}; return nil },
			Values:  rapid.Just("v"),
			Observe: func(_ *rapid.T, w *wkv) int { return len(w.data) },
			Allowed: func(_, _ int) bool { return false },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &wkv{}, &wkv{}); err == nil {
				rt.Fatal("expected disallowed transition to be flagged")
			}
		})
	})
}
