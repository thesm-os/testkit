// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"context"
	"errors"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
	"go.thesmos.sh/testkit/model/refappender"
	"go.thesmos.sh/testkit/model/refcas"
	"go.thesmos.sh/testkit/model/refcursor"
	"go.thesmos.sh/testkit/model/reflease"
	"go.thesmos.sh/testkit/model/refpool"
)

func TestAppenderMonotonicOffsets(t *testing.T) {
	t.Parallel()

	t.Run("monotonic appender passes", func(t *testing.T) {
		t.Parallel()
		log := refappender.NewMonotonicLog[string]()
		l := &law.AppenderMonotonicOffsets[*refappender.MonotonicLog[string], string, int64]{
			Append: func(rt *rapid.T, s *refappender.MonotonicLog[string], v string) (int64, error) {
				return s.Append(rt.Context(), v)
			},
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, log, log); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestCASAtomicOneWinner(t *testing.T) {
	t.Parallel()

	t.Run("refcas reference satisfies the law", func(t *testing.T) {
		t.Parallel()
		type entry struct {
			V       int
			Version int
		}
		errMismatch := errors.New("mismatch")

		l := law.CASAtomicOneWinner[*refcas.AtomicCell[entry, int], entry]{
			CAS: func(rt *rapid.T, c *refcas.AtomicCell[entry, int], e entry) error {
				return c.CompareAndSwap(rt.Context(), e)
			},
			Read: func(rt *rapid.T, c *refcas.AtomicCell[entry, int]) (entry, error) {
				v, ver, ok := c.Get(rt.Context())
				if !ok {
					var zero entry
					return zero, errors.New("empty")
				}
				v.Version = ver
				return v, nil
			},
			Values: rapid.SampledFrom([]entry{
				{V: 1, Version: 1},
				{V: 2, Version: 1},
			}),
			Mismatch: errMismatch,
		}

		rapid.Check(t, func(rt *rapid.T) {
			// Each iteration: fresh cell, seed v0 to advance to version 1,
			// then race two v1-versioned writes.
			c := refcas.NewAtomicCell(
				func(e entry) int { return e.Version },
				func(v int) int { return v + 1 },
				errMismatch,
			)
			_ = c.CompareAndSwap(rt.Context(), entry{V: 0, Version: 0}) // bootstrap → version=1
			if err := l.Check(rt, c, c); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestLeaseDoubleAcquireBlocks(t *testing.T) {
	t.Parallel()

	t.Run("reflease tracker satisfies the law", func(t *testing.T) {
		t.Parallel()
		errHeld := errors.New("held")
		errFree := errors.New("free")
		tr := reflease.NewTracker[string](errHeld, errFree)
		l := law.LeaseDoubleAcquireBlocks[*reflease.Tracker[string], string]{
			Acquire: func(rt *rapid.T, s *reflease.Tracker[string], k string) error {
				return s.Acquire(rt.Context(), k)
			},
			Release: func(rt *rapid.T, s *reflease.Tracker[string], k string) error {
				return s.Release(rt.Context(), k)
			},
			Keys: rapid.Just("k"),
			Held: errHeld,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, tr, tr); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestPoolBalancedGetPut(t *testing.T) {
	t.Parallel()

	t.Run("refpool reference reports balanced stats", func(t *testing.T) {
		t.Parallel()
		type conn struct{ id int }
		errDoublePut := errors.New("dp")
		next := 0
		p := refpool.NewBalancedPool(
			func() *conn { next++; return &conn{id: next} },
			func(c *conn) any { return c },
			errDoublePut,
		)
		l := law.PoolBalancedGetPut[*refpool.BalancedPool[*conn]]{
			Stats: func(_ *rapid.T, s *refpool.BalancedPool[*conn]) (int, int, int) {
				return s.Stats()
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, p, p); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestCursorCloseIdempotent(t *testing.T) {
	t.Parallel()

	t.Run("refcursor close-twice is idempotent", func(t *testing.T) {
		t.Parallel()
		errClosed := errors.New("closed")
		l := law.CursorCloseIdempotent[*refcursor.BoundedCursor[string]]{
			Close: func(rt *rapid.T, c *refcursor.BoundedCursor[string]) error {
				return c.Close(rt.Context())
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			c := refcursor.NewBoundedCursor([]string{"a"}, errClosed)
			if err := l.Check(rt, c, c); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

type persisterSUT struct {
	store map[int]string
	next  int
}

func (s *persisterSUT) save(_ *rapid.T, v string) (int, error) {
	if s.store == nil {
		s.store = make(map[int]string)
	}
	s.next++
	s.store[s.next] = v
	return s.next, nil
}

func (s *persisterSUT) read(_ *rapid.T, id int) (string, error) {
	v, ok := s.store[id]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func TestPersisterRetrievable(t *testing.T) {
	t.Parallel()

	t.Run("save→read returns the saved value", func(t *testing.T) {
		t.Parallel()
		s := &persisterSUT{}
		l := law.PersisterRetrievable[*persisterSUT, string, int]{
			Save:   func(rt *rapid.T, p *persisterSUT, v string) (int, error) { return p.save(rt, v) },
			Read:   func(rt *rapid.T, p *persisterSUT, id int) (string, error) { return p.read(rt, id) },
			Values: rapid.SampledFrom([]string{"a", "b", "c"}),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("read returning wrong value flagged", func(t *testing.T) {
		t.Parallel()
		s := &persisterSUT{}
		l := law.PersisterRetrievable[*persisterSUT, string, int]{
			Save:   func(rt *rapid.T, p *persisterSUT, v string) (int, error) { return p.save(rt, v) },
			Read:   func(_ *rapid.T, _ *persisterSUT, _ int) (string, error) { return "wrong", nil },
			Values: rapid.SampledFrom([]string{"a", "b"}),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected save/read mismatch")
			}
		})
	})
}

type updaterSUT struct {
	store map[string]string
}

func (s *updaterSUT) write(_ *rapid.T, v string) error {
	if s.store == nil {
		s.store = make(map[string]string)
	}
	s.store[v[:1]] = v // key = first byte
	return nil
}

func (s *updaterSUT) lookup(_ *rapid.T, k string) (string, error) {
	v, ok := s.store[k]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func TestUpdaterReplaces(t *testing.T) {
	t.Parallel()

	t.Run("last-write-wins under matching keys", func(t *testing.T) {
		t.Parallel()
		s := &updaterSUT{}
		l := law.UpdaterReplaces[*updaterSUT, string, string]{
			Update: func(rt *rapid.T, u *updaterSUT, v string) error { return u.write(rt, v) },
			Read:   func(rt *rapid.T, u *updaterSUT, k string) (string, error) { return u.lookup(rt, k) },
			Values: rapid.SampledFrom([]string{"a1", "a2", "b1", "b2"}),
			KeyOf:  func(v string) string { return v[:1] },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("non-replacing impl flagged", func(t *testing.T) {
		t.Parallel()
		s := &updaterSUT{store: map[string]string{"a": "stuck"}}
		l := law.UpdaterReplaces[*updaterSUT, string, string]{
			Update: func(_ *rapid.T, _ *updaterSUT, _ string) error { return nil }, // ignores writes
			Read:   func(rt *rapid.T, u *updaterSUT, k string) (string, error) { return u.lookup(rt, k) },
			Values: rapid.SampledFrom([]string{"a1", "a2"}),
			KeyOf:  func(v string) string { return v[:1] },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected replace failure")
			}
		})
	})
}

func TestUpserterIdempotent(t *testing.T) {
	t.Parallel()

	t.Run("repeated upsert of same value passes", func(t *testing.T) {
		t.Parallel()
		s := &updaterSUT{}
		l := law.UpserterIdempotent[*updaterSUT, string, string]{
			Upsert: func(rt *rapid.T, u *updaterSUT, v string) error { return u.write(rt, v) },
			Read:   func(rt *rapid.T, u *updaterSUT, k string) (string, error) { return u.lookup(rt, k) },
			Values: rapid.SampledFrom([]string{"a1", "b1"}),
			KeyOf:  func(v string) string { return v[:1] },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestTransactionRollbackOnError(t *testing.T) {
	t.Parallel()

	t.Run("rollback leaves probe key absent", func(t *testing.T) {
		t.Parallel()
		s := &updaterSUT{}
		notFound := errors.New("nf")
		l := law.TransactionRollbackOnError[*updaterSUT, string, string]{
			Run: func(_ *rapid.T, _ *updaterSUT, body func(context.Context) error) error {
				return body(t.Context())
			},
			Read: func(_ *rapid.T, u *updaterSUT, k string) (string, error) {
				if u.store == nil {
					return "", notFound
				}
				v, ok := u.store[k]
				if !ok {
					return "", notFound
				}
				return v, nil
			},
			Keys:     rapid.Just("a"),
			NotFound: notFound,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})
}
