// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"html"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

var xssVectors = rapid.SampledFrom([]string{
	"<script>alert(1)</script>",
	"<img src=x onerror=alert(1)>",
	"<svg onload=alert(1)>",
	"<iframe src=javascript:alert(1)>",
})

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

// wstore is a keyed store for the write-observable law.
type wstore struct {
	data map[string]string
	drop bool // when set, writes are silently dropped (the bug)
}

func (s *wstore) write(v string) error {
	if s.data == nil {
		s.data = make(map[string]string)
	}
	if s.drop {
		return nil
	}
	s.data[v] = v
	return nil
}

func (s *wstore) read(k string) (string, error) {
	v, ok := s.data[k]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func TestWriteObservable(t *testing.T) {
	t.Parallel()

	t.Run("written value is observable via read", func(t *testing.T) {
		t.Parallel()
		l := law.WriteObservable[*wstore, string, string]{
			Write:  func(_ *rapid.T, s *wstore, v string) error { return s.write(v) },
			Read:   func(_ *rapid.T, s *wstore, k string) (string, error) { return s.read(k) },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			KeyOf:  func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &wstore{}
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("store that drops writes is caught", func(t *testing.T) {
		t.Parallel()
		l := law.WriteObservable[*wstore, string, string]{
			Write:  func(_ *rapid.T, s *wstore, v string) error { return s.write(v) },
			Read:   func(_ *rapid.T, s *wstore, k string) (string, error) { return s.read(k) },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
			KeyOf:  func(v string) string { return v },
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &wstore{drop: true}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected dropped write to be caught")
			}
		})
	})
}

// tamperStore keeps a length checksum over its data. verify
// recomputes and compares — unless blind is set, the bug in which
// integrity is never checked. tamper corrupts data without updating
// the checksum.
type tamperStore struct {
	data  []string
	sum   int
	blind bool
}

func (s *tamperStore) write(v string) error {
	s.data = append(s.data, v)
	s.sum += len(v)
	return nil
}

func (s *tamperStore) verify() error {
	if s.blind {
		return nil
	}
	got := 0
	for _, v := range s.data {
		got += len(v)
	}
	if got != s.sum {
		return errors.New("integrity check failed")
	}
	return nil
}

func (s *tamperStore) tamper() bool {
	if len(s.data) == 0 {
		return false
	}
	s.data[0] += "X" // corrupt content without updating the checksum
	return true
}

func TestTamperEvident(t *testing.T) {
	t.Parallel()

	t.Run("checksum store detects post-write tampering", func(t *testing.T) {
		t.Parallel()
		l := law.TamperEvident[*tamperStore, string]{
			Write:  func(_ *rapid.T, s *tamperStore, v string) error { return s.write(v) },
			Tamper: func(_ *rapid.T, s *tamperStore) bool { return s.tamper() },
			Verify: func(_ *rapid.T, s *tamperStore) error { return s.verify() },
			Values: rapid.SampledFrom([]string{"a", "bb", "ccc"}),
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &tamperStore{}
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("store with no integrity check is caught", func(t *testing.T) {
		t.Parallel()
		l := law.TamperEvident[*tamperStore, string]{
			Write:  func(_ *rapid.T, s *tamperStore, v string) error { return s.write(v) },
			Tamper: func(_ *rapid.T, s *tamperStore) bool { return s.tamper() },
			Verify: func(_ *rapid.T, s *tamperStore) error { return s.verify() },
			Values: rapid.SampledFrom([]string{"a", "bb"}),
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &tamperStore{blind: true}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected undetected tampering to be caught")
			}
		})
	})
}

func TestXSSSafe(t *testing.T) {
	t.Parallel()

	t.Run("HTML-escaping renderer neutralizes XSS vectors", func(t *testing.T) {
		t.Parallel()
		l := law.XSSSafe[struct{}]{
			Render:   func(_ *rapid.T, _ struct{}, raw string) (string, error) { return html.EscapeString(raw), nil },
			Payloads: xssVectors,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("renderer that passes markup through verbatim is caught", func(t *testing.T) {
		t.Parallel()
		l := law.XSSSafe[struct{}]{
			Render:   func(_ *rapid.T, _ struct{}, raw string) (string, error) { return raw, nil }, // no escaping
			Payloads: xssVectors,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("expected unescaped markup to be caught")
			}
		})
	})
}

var injectionVectors = rapid.SampledFrom([]string{
	"' OR '1'='1",
	"'; DROP TABLE users; --",
	"admin'--",
	"$(rm -rf /)",
	"`whoami`",
})

// injStore stores values under keys. When vulnerable, a value
// containing shell/SQL metacharacters corrupts the canary — the
// injection "breaks out" of its parameter.
type injStore struct {
	data       map[string]string
	vulnerable bool
}

func (s *injStore) store(k, v string) error {
	if s.data == nil {
		s.data = make(map[string]string)
	}
	if s.vulnerable && strings.ContainsAny(v, "'\"`;$") {
		s.data["canary"] = "HACKED"
	}
	s.data[k] = v
	return nil
}

func (s *injStore) load(k string) (string, error) {
	v, ok := s.data[k]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func TestInjectionSafe(t *testing.T) {
	t.Parallel()

	t.Run("parameterized store keeps payloads as literal data", func(t *testing.T) {
		t.Parallel()
		l := law.InjectionSafe[*injStore]{
			Store:       func(_ *rapid.T, s *injStore, k, v string) error { return s.store(k, v) },
			Load:        func(_ *rapid.T, s *injStore, k string) (string, error) { return s.load(k) },
			Payloads:    injectionVectors,
			CanaryKey:   "canary",
			CanaryValue: "safe",
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &injStore{}
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("store where injection corrupts other data is caught", func(t *testing.T) {
		t.Parallel()
		l := law.InjectionSafe[*injStore]{
			Store:       func(_ *rapid.T, s *injStore, k, v string) error { return s.store(k, v) },
			Load:        func(_ *rapid.T, s *injStore, k string) (string, error) { return s.load(k) },
			Payloads:    injectionVectors,
			CanaryKey:   "canary",
			CanaryValue: "safe",
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &injStore{vulnerable: true}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected injection breakout to be caught")
			}
		})
	})
}
