// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/model/ref"
)

func TestAppenderMonotonicOffsets(t *testing.T) {
	t.Parallel()

	t.Run("monotonic appender passes", func(t *testing.T) {
		t.Parallel()
		log := ref.NewMonotonicLog[string]()
		l := &law.AppenderMonotonicOffsets[*ref.MonotonicLog[string], string, int64]{
			Append: func(rt *rapid.T, s *ref.MonotonicLog[string], v string) (int64, error) {
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

	t.Run("Reset clears the watermark for a fresh pair", func(t *testing.T) {
		t.Parallel()
		l := &law.AppenderMonotonicOffsets[*ref.MonotonicLog[string], string, int64]{
			Append: func(rt *rapid.T, s *ref.MonotonicLog[string], v string) (int64, error) {
				return s.Append(rt.Context(), v)
			},
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			grown := ref.NewMonotonicLog[string]()
			_, _ = grown.Append(rt.Context(), "seed")
			_, _ = grown.Append(rt.Context(), "seed")
			if err := l.Check(rt, grown, grown); err != nil {
				rt.Fatal(err)
			}
			l.Reset()
			// A fresh log answers offsets below the old watermark; only the
			// reset keeps that from reading as a violation.
			fresh := ref.NewMonotonicLog[string]()
			if err := l.Check(rt, fresh, fresh); err != nil {
				rt.Fatalf("the previous pair's offsets order nothing here: %v", err)
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

		l := law.CASAtomicOneWinner[*ref.AtomicCell[entry, int], entry]{
			CAS: func(rt *rapid.T, c *ref.AtomicCell[entry, int], e entry) error {
				return c.CompareAndSwap(rt.Context(), e)
			},
			Read: func(rt *rapid.T, c *ref.AtomicCell[entry, int]) (entry, error) {
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
			c := ref.NewAtomicCell(
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

	t.Run("the stamp makes stale draws coherent", func(t *testing.T) {
		t.Parallel()
		type entry struct {
			V       int
			Version int
		}
		errMismatch := errors.New("mismatch")

		l := law.CASAtomicOneWinner[*ref.AtomicCell[entry, int], entry]{
			CAS: func(rt *rapid.T, c *ref.AtomicCell[entry, int], e entry) error {
				return c.CompareAndSwap(rt.Context(), e)
			},
			Read: func(rt *rapid.T, c *ref.AtomicCell[entry, int]) (entry, error) {
				v, ver, ok := c.Get(rt.Context())
				if !ok {
					var zero entry
					return zero, errors.New("empty")
				}
				v.Version = ver
				return v, nil
			},
			// Deliberately stale draws: without the stamp both attempts
			// mismatch and the law reports no winner.
			Values: rapid.SampledFrom([]entry{
				{V: 1, Version: 99},
				{V: 2, Version: 99},
			}),
			Stamp: func(rt *rapid.T, c *ref.AtomicCell[entry, int], e entry) entry {
				_, ver, ok := c.Get(rt.Context())
				if !ok {
					e.Version = 0
					return e
				}
				e.Version = ver
				return e
			},
			Mismatch: errMismatch,
		}

		rapid.Check(t, func(rt *rapid.T) {
			c := ref.NewAtomicCell(
				func(e entry) int { return e.Version },
				func(v int) int { return v + 1 },
				errMismatch,
			)
			_ = c.CompareAndSwap(rt.Context(), entry{V: 0, Version: 0})
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
		tr := ref.NewLeaseTracker[string](errHeld, errFree)
		l := law.LeaseDoubleAcquireBlocks[*ref.LeaseTracker[string], string]{
			Acquire: func(rt *rapid.T, s *ref.LeaseTracker[string], k string) error {
				return s.Acquire(rt.Context(), k)
			},
			Release: func(rt *rapid.T, s *ref.LeaseTracker[string], k string) error {
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
		p := ref.NewBalancedPool(
			func() *conn { next++; return &conn{id: next} },
			func(c *conn) any { return c },
			errDoublePut,
		)
		l := law.PoolBalancedGetPut[*ref.BalancedPool[*conn]]{
			Stats: func(_ *rapid.T, s *ref.BalancedPool[*conn]) (int, int, int) {
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
		l := law.CursorCloseIdempotent[*ref.BoundedCursor[string]]{
			Close: func(rt *rapid.T, c *ref.BoundedCursor[string]) error {
				return c.Close(rt.Context())
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			c := ref.NewBoundedCursor([]string{"a"}, errClosed)
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

	// The first read succeeded, so a read that fails after the repeat is the
	// subject breaking rather than a key it never held.
	t.Run("a read that fails after the second upsert is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &updaterSUT{}
			reads := 0
			l := law.UpserterIdempotent[*updaterSUT, string, string]{
				Upsert: func(rt *rapid.T, u *updaterSUT, v string) error { return u.write(rt, v) },
				Read: func(rt *rapid.T, u *updaterSUT, k string) (string, error) {
					reads++
					if reads > 1 {
						return "", errors.New("index unavailable")
					}
					return u.lookup(rt, k)
				},
				Values: rapid.SampledFrom([]string{"a1", "b1"}),
				KeyOf:  func(v string) string { return v[:1] },
			}
			err := l.Check(rt, s, s)
			if err == nil || !strings.Contains(err.Error(), "read after second upsert errored") {
				rt.Fatalf("a read that stops answering must be reported, got: %v", err)
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

func TestPaginatorNoDuplicates(t *testing.T) {
	t.Parallel()

	t.Run("CursorTable walk emits each element exactly once", func(t *testing.T) {
		t.Parallel()
		tab := ref.NewCursorTable[int, int](func(a, b int) bool { return a < b })
		for i := range 10 {
			_ = tab.Put(t.Context(), i, i)
		}
		l := law.PaginatorNoDuplicates[*ref.CursorTable[int, int], int, int, int]{
			Page: func(rt *rapid.T, s *ref.CursorTable[int, int], cur int) ([]int, int, bool) {
				items, next, _ := s.Page(rt.Context(), cur, 3)
				return items, next, next != 0
			},
			Start:    0,
			KeyOf:    func(v int) int { return v },
			MaxPages: 100,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, tab, tab); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("paginator that re-emits the boundary element is caught", func(t *testing.T) {
		t.Parallel()
		data := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		// BUG: next cursor steps back one, re-emitting the last element
		// of each page as the first of the next.
		buggy := func(_ *rapid.T, _ struct{}, cur int) ([]int, int, bool) {
			if cur >= len(data) {
				return nil, 0, false
			}
			end := min(cur+3, len(data))
			next := end - 1
			return data[cur:end], next, end < len(data)
		}
		l := law.PaginatorNoDuplicates[struct{}, int, int, int]{
			Page:     buggy,
			Start:    0,
			KeyOf:    func(v int) int { return v },
			MaxPages: 100,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("expected duplicate-key detection across pages")
			}
		})
	})
}

// badPaginator advances an internal offset on each Page call and
// ignores the cursor it is handed — so it walks forward correctly
// once but cannot resume from a mid-stream cursor.
type badPaginator struct {
	data []int
	off  int
}

func TestPaginatorResumable(t *testing.T) {
	t.Parallel()

	t.Run("CursorTable resume equals the full-walk suffix", func(t *testing.T) {
		t.Parallel()
		tab := ref.NewCursorTable[int, int](func(a, b int) bool { return a < b })
		for i := range 10 {
			_ = tab.Put(t.Context(), i, i)
		}
		l := law.PaginatorResumable[*ref.CursorTable[int, int], int, int]{
			Page: func(rt *rapid.T, s *ref.CursorTable[int, int], cur int) ([]int, int, bool) {
				items, next, _ := s.Page(rt.Context(), cur, 3)
				return items, next, next != 0
			},
			Start:    0,
			MaxPages: 100,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, tab, tab); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("paginator that ignores the resume cursor is caught", func(t *testing.T) {
		t.Parallel()
		l := law.PaginatorResumable[*badPaginator, int, int]{
			Page: func(_ *rapid.T, s *badPaginator, _ int) ([]int, int, bool) {
				if s.off >= len(s.data) {
					return nil, 0, false
				}
				end := min(s.off+3, len(s.data))
				page := s.data[s.off:end]
				s.off = end // BUG: advances internal state, ignores the cursor
				return page, end, end < len(s.data)
			},
			Start:    0,
			MaxPages: 100,
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &badPaginator{data: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected non-resumable paginator to be caught")
			}
		})
	})

	// MaxPages ≤ 0 must not mean "walk zero pages", which would make every
	// paginator trivially resumable.
	t.Run("a non-positive page cap falls back to the default", func(t *testing.T) {
		t.Parallel()
		tab := ref.NewCursorTable[int, int](func(a, b int) bool { return a < b })
		for i := range 10 {
			_ = tab.Put(t.Context(), i, i)
		}
		pages := 0
		l := law.PaginatorResumable[*ref.CursorTable[int, int], int, int]{
			Page: func(rt *rapid.T, s *ref.CursorTable[int, int], cur int) ([]int, int, bool) {
				pages++
				items, next, _ := s.Page(rt.Context(), cur, 3)
				return items, next, next != 0
			},
			Start: 0,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, tab, tab); err != nil {
				rt.Fatalf("a resumable paginator must pass: %v", err)
			}
		})
		if pages == 0 {
			t.Fatal("MaxPages=0 must not collapse the walk to nothing")
		}
	})
}

func TestPublisherDelivers(t *testing.T) {
	t.Parallel()

	t.Run("at-least-once broker reaches every subscriber", func(t *testing.T) {
		t.Parallel()
		l := law.PublisherDelivers[*ref.AtLeastOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *ref.AtLeastOnce[int]) (int, error) {
				return s.Subscribe(rt.Context())
			},
			Publish: func(rt *rapid.T, s *ref.AtLeastOnce[int], m int) error {
				return s.Publish(rt.Context(), m)
			},
			Drain: func(rt *rapid.T, s *ref.AtLeastOnce[int], sub int) ([]int, error) {
				return s.Drain(rt.Context(), sub)
			},
			Messages:    rapid.Int(),
			Subscribers: 3,
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := ref.NewAtLeastOnce[int]()
			if err := l.Check(rt, b, ref.NewAtLeastOnce[int]()); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("broker that drops everything is caught", func(t *testing.T) {
		t.Parallel()
		l := law.PublisherDelivers[*ref.AtMostOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *ref.AtMostOnce[int]) (int, error) {
				return s.Subscribe(rt.Context())
			},
			Publish: func(rt *rapid.T, s *ref.AtMostOnce[int], m int) error {
				return s.Publish(rt.Context(), m)
			},
			Drain: func(rt *rapid.T, s *ref.AtMostOnce[int], sub int) ([]int, error) {
				msgs, _, err := s.Drain(rt.Context(), sub)
				return msgs, err
			},
			Messages:    rapid.Int(),
			Subscribers: 3,
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := ref.NewAtMostOnce[int](0) // capacity 0 → drops all
			if err := l.Check(rt, b, b); err == nil {
				rt.Fatal("expected non-delivery to be caught")
			}
		})
	})
}

func TestPublisherDeliveryBound(t *testing.T) {
	t.Parallel()

	t.Run("at-least-once: redelivered message counts >= 1", func(t *testing.T) {
		t.Parallel()
		l := law.PublisherDeliveryBound[*ref.AtLeastOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *ref.AtLeastOnce[int]) (int, error) { return s.Subscribe(rt.Context()) },
			Publish:   func(rt *rapid.T, s *ref.AtLeastOnce[int], m int) error { return s.Publish(rt.Context(), m) },
			Redeliver: func(rt *rapid.T, s *ref.AtLeastOnce[int], m int) { _ = s.Publish(rt.Context(), m) },
			Drain: func(rt *rapid.T, s *ref.AtLeastOnce[int], sub int) ([]int, error) {
				return s.Drain(rt.Context(), sub)
			},
			Messages: rapid.Int(),
			Mode:     law.DeliveryAtLeastOnce,
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := ref.NewAtLeastOnce[int]()
			if err := l.Check(rt, b, ref.NewAtLeastOnce[int]()); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("at-most-once: single publish counts <= 1", func(t *testing.T) {
		t.Parallel()
		l := law.PublisherDeliveryBound[*ref.AtMostOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *ref.AtMostOnce[int]) (int, error) { return s.Subscribe(rt.Context()) },
			Publish:   func(rt *rapid.T, s *ref.AtMostOnce[int], m int) error { return s.Publish(rt.Context(), m) },
			Drain: func(rt *rapid.T, s *ref.AtMostOnce[int], sub int) ([]int, error) {
				msgs, _, err := s.Drain(rt.Context(), sub)
				return msgs, err
			},
			Messages: rapid.Int(),
			Mode:     law.DeliveryAtMostOnce,
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := ref.NewAtMostOnce[int](4)
			if err := l.Check(rt, b, ref.NewAtMostOnce[int](4)); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("exactly-once: replay of same id counts == 1", func(t *testing.T) {
		t.Parallel()
		var lastID int64
		l := law.PublisherDeliveryBound[*ref.ExactlyOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *ref.ExactlyOnce[int]) (int, error) { return s.Subscribe(rt.Context()) },
			Publish: func(rt *rapid.T, s *ref.ExactlyOnce[int], m int) error {
				id, err := s.Publish(rt.Context(), m)
				lastID = id
				return err
			},
			Redeliver: func(rt *rapid.T, s *ref.ExactlyOnce[int], m int) { _ = s.Replay(rt.Context(), lastID, m) },
			Drain: func(rt *rapid.T, s *ref.ExactlyOnce[int], sub int) ([]int, error) {
				return s.Drain(rt.Context(), sub)
			},
			Messages: rapid.Int(),
			Mode:     law.DeliveryExactlyOnce,
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := ref.NewExactlyOnce[int]()
			if err := l.Check(rt, b, ref.NewExactlyOnce[int]()); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("exactly-once mode catches a broker that duplicates", func(t *testing.T) {
		t.Parallel()
		l := law.PublisherDeliveryBound[*ref.AtLeastOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *ref.AtLeastOnce[int]) (int, error) { return s.Subscribe(rt.Context()) },
			Publish:   func(rt *rapid.T, s *ref.AtLeastOnce[int], m int) error { return s.Publish(rt.Context(), m) },
			Redeliver: func(rt *rapid.T, s *ref.AtLeastOnce[int], m int) { _ = s.Publish(rt.Context(), m) }, // duplicates
			Drain: func(rt *rapid.T, s *ref.AtLeastOnce[int], sub int) ([]int, error) {
				return s.Drain(rt.Context(), sub)
			},
			Messages: rapid.Int(),
			Mode:     law.DeliveryExactlyOnce, // but broker duplicates → must fire
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := ref.NewAtLeastOnce[int]()
			if err := l.Check(rt, b, b); err == nil {
				rt.Fatal("expected exactly-once violation under duplicating broker")
			}
		})
	})
}

// leakyTxStore is a broken "transactional" store whose tx writes
// land in the shared map immediately — no isolation.
type (
	leakyTxStore struct{ data map[string]int }
	leakyTx      struct{ store *leakyTxStore }
)

func TestTransactionNoMidTxVisibility(t *testing.T) {
	t.Parallel()

	t.Run("snapshot-isolation hides uncommitted writes", func(t *testing.T) {
		t.Parallel()
		errNF := errors.New("not found")
		l := law.TransactionNoMidTxVisibility[*ref.SnapshotIsolation[string, int], *ref.SnapshotTx[string, int], string, int]{
			Begin: func(rt *rapid.T, s *ref.SnapshotIsolation[string, int]) (*ref.SnapshotTx[string, int], error) {
				return s.Begin(rt.Context())
			},
			TxPut: func(rt *rapid.T, _ *ref.SnapshotIsolation[string, int], tx *ref.SnapshotTx[string, int], k string, v int) error {
				return tx.Put(rt.Context(), k, v)
			},
			TxRollback: func(rt *rapid.T, _ *ref.SnapshotIsolation[string, int], tx *ref.SnapshotTx[string, int]) error {
				return tx.Rollback(rt.Context())
			},
			Read: func(rt *rapid.T, s *ref.SnapshotIsolation[string, int], k string) (int, error) {
				return s.Get(rt.Context(), k)
			},
			Keys:   rapid.SampledFrom([]string{"a", "b"}),
			Values: rapid.Int(),
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := ref.NewSnapshotIsolation[string, int](errNF)
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("store that leaks uncommitted writes is caught", func(t *testing.T) {
		t.Parallel()
		errNF := errors.New("not found")
		l := law.TransactionNoMidTxVisibility[*leakyTxStore, *leakyTx, string, int]{
			Begin: func(_ *rapid.T, s *leakyTxStore) (*leakyTx, error) { return &leakyTx{store: s}, nil },
			TxPut: func(_ *rapid.T, _ *leakyTxStore, tx *leakyTx, k string, v int) error {
				tx.store.data[k] = v
				return nil
			}, // BUG: writes through
			TxRollback: func(_ *rapid.T, _ *leakyTxStore, _ *leakyTx) error { return nil },
			Read: func(_ *rapid.T, s *leakyTxStore, k string) (int, error) {
				v, ok := s.data[k]
				if !ok {
					return 0, errNF
				}
				return v, nil
			},
			Keys:   rapid.SampledFrom([]string{"a", "b"}),
			Values: rapid.Int(),
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := &leakyTxStore{data: make(map[string]int)}
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected mid-tx visibility to be caught")
			}
		})
	})
}

// looseTx is a broken two-phase transaction: both Commit and
// Rollback always succeed, violating the commit-XOR-rollback mutex.
type looseTx struct{}

func TestTwoPhaseCommitOrRollback(t *testing.T) {
	t.Parallel()

	t.Run("snapshot-isolation tx commits XOR rolls back", func(t *testing.T) {
		t.Parallel()
		l := law.TwoPhaseCommitOrRollback[*ref.SnapshotIsolation[string, int], *ref.SnapshotTx[string, int]]{
			Begin: func(rt *rapid.T, s *ref.SnapshotIsolation[string, int]) (*ref.SnapshotTx[string, int], error) {
				return s.Begin(rt.Context())
			},
			Commit: func(rt *rapid.T, _ *ref.SnapshotIsolation[string, int], tx *ref.SnapshotTx[string, int]) error {
				return tx.Commit(rt.Context())
			},
			Rollback: func(rt *rapid.T, _ *ref.SnapshotIsolation[string, int], tx *ref.SnapshotTx[string, int]) error {
				return tx.Rollback(rt.Context())
			},
			Closed: ref.ErrTxClosed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := ref.NewSnapshotIsolation[string, int](errors.New("nf"))
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("tx that allows both commit and rollback is caught", func(t *testing.T) {
		t.Parallel()
		errClosed := errors.New("closed")
		l := law.TwoPhaseCommitOrRollback[struct{}, *looseTx]{
			Begin:    func(_ *rapid.T, _ struct{}) (*looseTx, error) { return &looseTx{}, nil },
			Commit:   func(_ *rapid.T, _ struct{}, _ *looseTx) error { return nil },
			Rollback: func(_ *rapid.T, _ struct{}, _ *looseTx) error { return nil }, // BUG: no mutex
			Closed:   errClosed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("expected commit/rollback mutex violation")
			}
		})
	})
}

var errLeaseHeld = errors.New("lease held")

// ctxLease is a context-bound lease fixture. When honorCancel is
// set it releases a key the moment the acquiring context is
// cancelled; otherwise it ignores cancellation entirely (the bug).
type ctxLease struct {
	mu          sync.Mutex
	held        map[string]bool
	honorCancel bool
}

func newCtxLease(honor bool) *ctxLease {
	return &ctxLease{held: map[string]bool{}, honorCancel: honor}
}

func (l *ctxLease) acquire(ctx context.Context, k string) error {
	l.mu.Lock()
	if l.held[k] {
		l.mu.Unlock()
		return errLeaseHeld
	}
	l.held[k] = true
	l.mu.Unlock()
	if l.honorCancel {
		go func() {
			<-ctx.Done()
			l.mu.Lock()
			delete(l.held, k)
			l.mu.Unlock()
		}()
	}
	return nil
}

func (l *ctxLease) free(k string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return !l.held[k]
}

func TestLeaseReleasedOnCancel(t *testing.T) {
	t.Parallel()

	t.Run("lease that honors cancel is released", func(t *testing.T) {
		t.Parallel()
		l := law.LeaseReleasedOnCancel[*ctxLease, string]{
			Acquire: func(ctx context.Context, s *ctxLease, k string) error { return s.acquire(ctx, k) },
			Free:    func(_ *rapid.T, s *ctxLease, k string) bool { return s.free(k) },
			Keys:    rapid.SampledFrom([]string{"a", "b"}),
			Timeout: time.Second,
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newCtxLease(true)
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("lease that ignores cancel is caught", func(t *testing.T) {
		t.Parallel()
		l := law.LeaseReleasedOnCancel[*ctxLease, string]{
			Acquire: func(ctx context.Context, s *ctxLease, k string) error { return s.acquire(ctx, k) },
			Free:    func(_ *rapid.T, s *ctxLease, k string) bool { return s.free(k) },
			Keys:    rapid.SampledFrom([]string{"a", "b"}),
			Timeout: 5 * time.Millisecond,
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newCtxLease(false)
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected lease-not-released-on-cancel to be caught")
			}
		})
	})
}

// watchable is a minimal change-notification fixture. watch returns
// a buffered channel registered for a key; mutate fans a value out
// to every channel watching that key — unless silent is set, the
// bug in which mutations are observed by nobody.
type watchable struct {
	mu     sync.Mutex
	subs   map[string][]chan int
	silent bool
}

func newWatchable(silent bool) *watchable {
	return &watchable{subs: map[string][]chan int{}, silent: silent}
}

func (w *watchable) watch(k string) chan int {
	ch := make(chan int, 4)
	w.mu.Lock()
	w.subs[k] = append(w.subs[k], ch)
	w.mu.Unlock()
	return ch
}

func (w *watchable) mutate(k string, v int) {
	if w.silent {
		return
	}
	w.mu.Lock()
	chs := append([]chan int(nil), w.subs[k]...)
	w.mu.Unlock()
	for _, ch := range chs {
		select {
		case ch <- v:
		default:
		}
	}
}

func nextWatch(ch chan int, timeout time.Duration) (int, bool) {
	select {
	case v := <-ch:
		return v, true
	case <-time.After(timeout):
		return 0, false
	}
}

func TestWatcherReturnsOnChange(t *testing.T) {
	t.Parallel()

	t.Run("watch established before a change observes it", func(t *testing.T) {
		t.Parallel()
		l := law.WatcherReturnsOnChange[*watchable, chan int, string, int]{
			Watch:   func(_ *rapid.T, s *watchable, k string) (chan int, error) { return s.watch(k), nil },
			Mutate:  func(_ *rapid.T, s *watchable, k string, v int) error { s.mutate(k, v); return nil },
			Next:    nextWatch,
			Stop:    func(_ chan int) {},
			Keys:    rapid.SampledFrom([]string{"a", "b"}),
			Values:  rapid.Int(),
			Timeout: time.Second,
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newWatchable(false)
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("watcher that never fires is caught", func(t *testing.T) {
		t.Parallel()
		l := law.WatcherReturnsOnChange[*watchable, chan int, string, int]{
			Watch:   func(_ *rapid.T, s *watchable, k string) (chan int, error) { return s.watch(k), nil },
			Mutate:  func(_ *rapid.T, s *watchable, k string, v int) error { s.mutate(k, v); return nil },
			Next:    nextWatch,
			Stop:    func(_ chan int) {},
			Keys:    rapid.SampledFrom([]string{"a", "b"}),
			Values:  rapid.Int(),
			Timeout: 10 * time.Millisecond,
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newWatchable(true) // silent
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected non-firing watcher to be caught")
			}
		})
	})

	t.Run("a refused watch holds vacuously", func(t *testing.T) {
		t.Parallel()
		l := law.WatcherReturnsOnChange[*watchable, chan int, string, int]{
			Watch:   func(*rapid.T, *watchable, string) (chan int, error) { return nil, errors.New("no watches") },
			Mutate:  func(_ *rapid.T, s *watchable, k string, v int) error { s.mutate(k, v); return nil },
			Next:    nextWatch,
			Stop:    func(chan int) {},
			Keys:    rapid.SampledFrom([]string{"a", "b"}),
			Values:  rapid.Int(),
			Timeout: 10 * time.Millisecond,
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newWatchable(false)
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a subject that cannot watch is a precondition: %v", err)
			}
		})
	})

	t.Run("a refused mutation holds vacuously", func(t *testing.T) {
		t.Parallel()
		l := law.WatcherReturnsOnChange[*watchable, chan int, string, int]{
			Watch:   func(_ *rapid.T, s *watchable, k string) (chan int, error) { return s.watch(k), nil },
			Mutate:  func(*rapid.T, *watchable, string, int) error { return errors.New("read-only") },
			Next:    nextWatch,
			Stop:    func(chan int) {},
			Keys:    rapid.SampledFrom([]string{"a", "b"}),
			Values:  rapid.Int(),
			Timeout: 10 * time.Millisecond,
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newWatchable(false)
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a subject that refuses the change is a precondition: %v", err)
			}
		})
	})

	// Timeout ≤ 0 must not mean "wait no time at all", which would flag every
	// watcher as silent.
	t.Run("a non-positive timeout falls back to the default", func(t *testing.T) {
		t.Parallel()
		l := law.WatcherReturnsOnChange[*watchable, chan int, string, int]{
			Watch:  func(_ *rapid.T, s *watchable, k string) (chan int, error) { return s.watch(k), nil },
			Mutate: func(_ *rapid.T, s *watchable, k string, v int) error { s.mutate(k, v); return nil },
			Next:   nextWatch,
			Stop:   func(chan int) {},
			Keys:   rapid.SampledFrom([]string{"a", "b"}),
			Values: rapid.Int(),
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := newWatchable(false)
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatalf("a firing watcher must pass under the default window: %v", err)
			}
		})
	})
}

// countingSingleflight is a deliberately broken single-flight: it never
// coalesces, so every concurrent call runs compute. The law must notice.
type countingSingleflight struct {
	mu sync.Mutex
	n  *int
}

func (s *countingSingleflight) call(_ context.Context, _ string, fn func() string) (string, error) {
	s.mu.Lock()
	*s.n++
	s.mu.Unlock()
	return fn(), nil
}

func TestSingleflightCoalesces(t *testing.T) {
	t.Parallel()

	t.Run("a coalescing implementation passes", func(t *testing.T) {
		t.Parallel()
		counter := 0
		var mu sync.Mutex
		c := ref.NewCoalescer[string, string]()
		l := law.SingleflightCoalesces[*ref.Coalescer[string, string], string, string]{
			Call: func(ctx context.Context, s *ref.Coalescer[string, string], k string, compute func() string) (string, error) {
				return s.Do(ctx, k, compute)
			},
			Compute: func() string {
				mu.Lock()
				counter++
				mu.Unlock()
				return "v"
			},
			Keys:     rapid.Just("k"),
			Parallel: 4,
			Counter:  &counter,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, c, c); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("an implementation that never coalesces is flagged", func(t *testing.T) {
		t.Parallel()
		counter := 0
		s := &countingSingleflight{n: &counter}
		l := law.SingleflightCoalesces[*countingSingleflight, string, string]{
			Call: func(ctx context.Context, sut *countingSingleflight, k string, compute func() string) (string, error) {
				return sut.call(ctx, k, compute)
			},
			Compute:  func() string { return "v" },
			Keys:     rapid.Just("k"),
			Parallel: 4,
			Counter:  &counter,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected a coalescing violation")
			}
		})
	})

	// Parallel <= 0 selects the default fan-out rather than launching no
	// goroutines, which would make the law vacuous.
	t.Run("non-positive Parallel falls back to the default fan-out", func(t *testing.T) {
		t.Parallel()
		counter := 0
		s := &countingSingleflight{n: &counter}
		l := law.SingleflightCoalesces[*countingSingleflight, string, string]{
			Call: func(ctx context.Context, sut *countingSingleflight, k string, compute func() string) (string, error) {
				return sut.call(ctx, k, compute)
			},
			Compute:  func() string { return "v" },
			Keys:     rapid.Just("k"),
			Parallel: 0,
			Counter:  &counter,
		}
		rapid.Check(t, func(rt *rapid.T) {
			before := counter
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected a coalescing violation")
			}
			if counter-before < 2 {
				rt.Fatalf("default fan-out must launch several calls, got %d", counter-before)
			}
		})
	})

	t.Run("identity", func(t *testing.T) {
		t.Parallel()
		var l law.SingleflightCoalesces[*countingSingleflight, string, string]
		if l.ID() != "AUTO-SINGLEFLIGHT-COALESCES" {
			t.Fatalf("unexpected law ID %q", l.ID())
		}
		if l.REQID() != "" {
			t.Fatalf("auto-derived laws carry no REQ tag, got %q", l.REQID())
		}
	})
}

// Every contract law distinguishes a failed precondition — the subject
// refused the setup call, so the law holds vacuously and returns nil — from a
// genuine violation, which returns a diagnostic. Confusing the two makes a law
// either silently useless or permanently red, so both paths are driven here
// with stubs that fail on a chosen call.

// failOnNth returns an error the nth time it is called (1-based), nil
// otherwise. It exists so a test can fail exactly one step of a multi-step law.
func failOnNth(n int) func() error {
	calls := 0
	var mu sync.Mutex
	return func() error {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls == n {
			return errors.New("injected failure")
		}
		return nil
	}
}

func TestContractLawPreconditionsAndViolations(t *testing.T) {
	t.Parallel()

	t.Run("PersisterRetrievable holds vacuously when Save errors", func(t *testing.T) {
		t.Parallel()
		l := law.PersisterRetrievable[*persisterSUT, string, int]{
			Save:   func(*rapid.T, *persisterSUT, string) (int, error) { return 0, errors.New("no") },
			Read:   func(*rapid.T, *persisterSUT, int) (string, error) { return "", nil },
			Values: rapid.Just("v"),
		}
		s := &persisterSUT{}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a refused Save is a precondition, not a violation: %v", err)
			}
		})
	})

	t.Run("PersisterRetrievable flags a Read error after a successful Save", func(t *testing.T) {
		t.Parallel()
		s := &persisterSUT{}
		l := law.PersisterRetrievable[*persisterSUT, string, int]{
			Save:   func(rt *rapid.T, p *persisterSUT, v string) (int, error) { return p.save(rt, v) },
			Read:   func(*rapid.T, *persisterSUT, int) (string, error) { return "", errors.New("gone") },
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a saved id that cannot be read is a violation")
			}
		})
	})

	t.Run("UpdaterReplaces holds vacuously when a write errors", func(t *testing.T) {
		t.Parallel()
		for _, nth := range []int{1, 2} {
			rapid.Check(t, func(rt *rapid.T) {
				// Built inside the property: rapid runs this many times, so a
				// counter shared across iterations would only fail the first.
				s := &updaterSUT{}
				fail := failOnNth(nth)
				l := law.UpdaterReplaces[*updaterSUT, string, string]{
					// The failure targets the subject's writes: a refusal on
					// the mirror is the law's report, not a precondition, and
					// has its own subtest in the pair block.
					Update: func(_ *rapid.T, u *updaterSUT, _ string) error {
						if u == s {
							return fail()
						}
						return nil
					},
					Read:   func(rt *rapid.T, u *updaterSUT, k string) (string, error) { return u.lookup(rt, k) },
					Values: rapid.Just("aa"),
					KeyOf:  func(v string) string { return v[:1] },
				}
				if err := l.Check(rt, s, &updaterSUT{}); !law.Holds(err) {
					rt.Fatalf("a refused write is a precondition, not a violation: %v", err)
				}
			})
		}
	})

	t.Run("UpdaterReplaces flags a Read error after both writes", func(t *testing.T) {
		t.Parallel()
		s := &updaterSUT{}
		l := law.UpdaterReplaces[*updaterSUT, string, string]{
			Update: func(rt *rapid.T, u *updaterSUT, v string) error { return u.write(rt, v) },
			Read:   func(*rapid.T, *updaterSUT, string) (string, error) { return "", errors.New("gone") },
			Values: rapid.Just("aa"),
			KeyOf:  func(v string) string { return v[:1] },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("an unreadable key after two writes is a violation")
			}
		})
	})

	t.Run("UpserterIdempotent separates preconditions from violations", func(t *testing.T) {
		t.Parallel()
		s := &updaterSUT{}
		mk := func(upsert func(*rapid.T, *updaterSUT, string) error,
			read func(*rapid.T, *updaterSUT, string) (string, error),
		) law.UpserterIdempotent[*updaterSUT, string, string] {
			return law.UpserterIdempotent[*updaterSUT, string, string]{
				Upsert: upsert, Read: read,
				Values: rapid.Just("aa"),
				KeyOf:  func(v string) string { return v[:1] },
			}
		}
		okRead := func(rt *rapid.T, u *updaterSUT, k string) (string, error) { return u.lookup(rt, k) }
		okUpsert := func(rt *rapid.T, u *updaterSUT, v string) error { return u.write(rt, v) }

		// First upsert refused, and first read refused: both vacuous. The
		// failing stub is rebuilt per iteration — rapid runs the property many
		// times, and a shared counter would only fail the first pass.
		rapid.Check(t, func(rt *rapid.T) {
			fail := failOnNth(1)
			l := mk(func(*rapid.T, *updaterSUT, string) error { return fail() }, okRead)
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a refused first upsert must hold vacuously: %v", err)
			}
		})
		rapid.Check(t, func(rt *rapid.T) {
			l := mk(okUpsert, func(*rapid.T, *updaterSUT, string) (string, error) {
				return "", errors.New("not yet")
			})
			if err := l.Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a refused first read must hold vacuously: %v", err)
			}
		})

		// Second upsert refused: the subject accepted the value once and then
		// rejected an identical write, which is the violation.
		rapid.Check(t, func(rt *rapid.T) {
			fail := failOnNth(2)
			l := mk(func(*rapid.T, *updaterSUT, string) error { return fail() }, okRead)
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a refused second upsert is a violation")
			}
		})
	})

	t.Run("UpserterIdempotent flags an unstable read", func(t *testing.T) {
		t.Parallel()
		s := &updaterSUT{}
		reads := 0
		l := law.UpserterIdempotent[*updaterSUT, string, string]{
			Upsert: func(rt *rapid.T, u *updaterSUT, v string) error { return u.write(rt, v) },
			Read: func(*rapid.T, *updaterSUT, string) (string, error) {
				reads++
				return string(rune('a' + reads)), nil
			},
			Values: rapid.Just("aa"),
			KeyOf:  func(v string) string { return v[:1] },
		}
		rapid.Check(t, func(rt *rapid.T) {
			reads = 0
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a read that changes after an idempotent upsert is a violation")
			}
		})
	})

	t.Run("AppenderMonotonicOffsets holds vacuously when Append errors", func(t *testing.T) {
		t.Parallel()
		log := ref.NewMonotonicLog[string]()
		l := &law.AppenderMonotonicOffsets[*ref.MonotonicLog[string], string, int64]{
			Append: func(*rapid.T, *ref.MonotonicLog[string], string) (int64, error) {
				return 0, errors.New("closed")
			},
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, log, log); !law.Holds(err) {
				rt.Fatalf("a refused Append is a precondition, not a violation: %v", err)
			}
		})
	})

	t.Run("AppenderMonotonicOffsets flags a non-advancing offset", func(t *testing.T) {
		t.Parallel()
		log := ref.NewMonotonicLog[string]()
		l := &law.AppenderMonotonicOffsets[*ref.MonotonicLog[string], string, int64]{
			Append: func(*rapid.T, *ref.MonotonicLog[string], string) (int64, error) {
				return 7, nil // constant offset: never advances
			},
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			_ = l.Check(rt, log, log) // first call primes prev
			if err := l.Check(rt, log, log); err == nil {
				rt.Fatal("a repeated offset is a monotonicity violation")
			}
		})
	})
}

// CAS is the one law here with no vacuous path: two concurrent writes against
// the same version must produce exactly one winner and one mismatch. Any other
// combination — both succeeding, both failing, or failing with the wrong error
// — is a violation, so each is driven separately.
func TestCASAtomicOneWinnerOutcomes(t *testing.T) {
	t.Parallel()

	mismatch := errors.New("version mismatch")
	mk := func(cas func(*rapid.T, int, string) error) law.CASAtomicOneWinner[int, string] {
		return law.CASAtomicOneWinner[int, string]{
			CAS: cas, Values: rapid.Just("v"), Mismatch: mismatch,
		}
	}

	t.Run("one winner and one mismatch passes", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			calls := 0
			l := mk(func(*rapid.T, int, string) error {
				calls++
				if calls == 1 {
					return nil
				}
				return mismatch
			})
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("exactly one winner is the expected outcome: %v", err)
			}
		})
	})

	t.Run("both succeeding is a violation", func(t *testing.T) {
		t.Parallel()
		l := mk(func(*rapid.T, int, string) error { return nil })
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("two winners means the CAS is not atomic")
			}
		})
	})

	t.Run("both failing is a violation", func(t *testing.T) {
		t.Parallel()
		l := mk(func(*rapid.T, int, string) error { return mismatch })
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("no winner means the CAS rejected a valid write")
			}
		})
	})

	// A loser that fails with some other error is as wrong as one that
	// succeeds: the caller cannot distinguish contention from a real fault.
	t.Run("the loser must fail with the mismatch sentinel", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			calls := 0
			l := mk(func(*rapid.T, int, string) error {
				calls++
				if calls == 1 {
					return nil
				}
				return errors.New("io error")
			})
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("the loser must report the mismatch sentinel")
			}
		})
	})
}

// A rolled-back transaction must leave the store observationally identical —
// both in what it holds and in whether the probed key exists at all.
func TestTransactionRollbackOnErrorOutcomes(t *testing.T) {
	t.Parallel()

	mk := func(read func(*rapid.T, *kvStore, string) (string, error)) law.TransactionRollbackOnError[*kvStore, string, string] {
		return law.TransactionRollbackOnError[*kvStore, string, string]{
			Run: func(rt *rapid.T, _ *kvStore, body func(context.Context) error) error {
				return body(rt.Context())
			},
			Read: read,
			Keys: rapid.Just("k"),
		}
	}

	t.Run("a clean rollback passes", func(t *testing.T) {
		t.Parallel()
		s := newKVStore()
		l := mk(func(_ *rapid.T, st *kvStore, k string) (string, error) { return st.get(k) })
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatalf("a store unchanged by a failed body must pass: %v", err)
			}
		})
	})

	t.Run("a key materialising across the rollback is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			reads := 0
			s := newKVStore()
			l := mk(func(_ *rapid.T, st *kvStore, k string) (string, error) {
				reads++
				if reads == 1 {
					return "", errors.New("absent")
				}
				return "leaked", nil
			})
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a key that appears after a rollback is a violation")
			}
		})
	})

	t.Run("a value changed across the rollback is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			reads := 0
			s := newKVStore()
			l := mk(func(*rapid.T, *kvStore, string) (string, error) {
				reads++
				return string(rune('a' + reads)), nil
			})
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("a value mutated by a rolled-back body is a violation")
			}
		})
	})
}

func TestLeaseDoubleAcquireBlocksOutcomes(t *testing.T) {
	t.Parallel()

	held := errors.New("already held")

	t.Run("a refused first acquire holds vacuously", func(t *testing.T) {
		t.Parallel()
		l := law.LeaseDoubleAcquireBlocks[int, string]{
			Acquire: func(*rapid.T, int, string) error { return errors.New("unavailable") },
			Release: func(*rapid.T, int, string) error { return nil },
			Keys:    rapid.Just("k"),
			Held:    held,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); !law.Holds(err) {
				rt.Fatalf("a lease that cannot be taken is a precondition: %v", err)
			}
		})
	})

	t.Run("a second acquire that succeeds is a violation", func(t *testing.T) {
		t.Parallel()
		l := law.LeaseDoubleAcquireBlocks[int, string]{
			Acquire: func(*rapid.T, int, string) error { return nil }, // never blocks
			Release: func(*rapid.T, int, string) error { return nil },
			Keys:    rapid.Just("k"),
			Held:    held,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("a lease that can be taken twice does not exclude")
			}
		})
	})

	t.Run("a second acquire failing with the wrong error is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			calls := 0
			l := law.LeaseDoubleAcquireBlocks[int, string]{
				Acquire: func(*rapid.T, int, string) error {
					calls++
					if calls == 1 {
						return nil
					}
					return errors.New("io")
				},
				Release: func(*rapid.T, int, string) error { return nil },
				Keys:    rapid.Just("k"),
				Held:    held,
			}
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("the blocked acquire must report the held sentinel")
			}
		})
	})
}

// A paginated walk has two failure modes the law must separate: emitting the
// same element on two pages, and never terminating. The page cap is what turns
// the second from a hang into a diagnostic.
func TestPaginatorNoDuplicatesOutcomes(t *testing.T) {
	t.Parallel()

	type pager struct{ pages [][]int }
	mk := func(p *pager, maxPages int) law.PaginatorNoDuplicates[*pager, int, int, int] {
		return law.PaginatorNoDuplicates[*pager, int, int, int]{
			Page: func(_ *rapid.T, s *pager, c int) ([]int, int, bool) {
				if c >= len(s.pages) {
					return nil, c, false
				}
				return s.pages[c], c + 1, c+1 < len(s.pages)
			},
			KeyOf:    func(v int) int { return v },
			Start:    0,
			MaxPages: maxPages,
		}
	}

	t.Run("a walk with distinct elements passes", func(t *testing.T) {
		t.Parallel()
		p := &pager{pages: [][]int{{1, 2}, {3, 4}}}
		l := mk(p, 10)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, p, p); err != nil {
				rt.Fatalf("a duplicate-free walk must pass: %v", err)
			}
		})
	})

	t.Run("an element on two pages is a violation", func(t *testing.T) {
		t.Parallel()
		p := &pager{pages: [][]int{{1, 2}, {2, 3}}}
		l := mk(p, 10)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, p, p); err == nil {
				rt.Fatal("an element repeated across pages is a violation")
			}
		})
	})

	// A pager that always reports more pages would loop forever without the
	// cap, so the law must stop and say so.
	t.Run("a non-terminating walk is caught by the page cap", func(t *testing.T) {
		t.Parallel()
		l := law.PaginatorNoDuplicates[*pager, int, int, int]{
			Page: func(_ *rapid.T, _ *pager, c int) ([]int, int, bool) {
				return []int{c}, c + 1, true // never done
			},
			KeyOf:    func(v int) int { return v },
			Start:    0,
			MaxPages: 5,
		}
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, nil, nil)
			if err == nil {
				rt.Fatal("a walk that never ends is a violation")
			}
			if !strings.Contains(err.Error(), "terminate") {
				rt.Fatalf("the diagnostic must name the cause, got: %v", err)
			}
		})
	})

	// MaxPages <= 0 selects a generous default rather than capping at zero,
	// which would fail every paginator immediately.
	t.Run("a non-positive page cap falls back to a default", func(t *testing.T) {
		t.Parallel()
		p := &pager{pages: [][]int{{1}, {2}}}
		l := mk(p, 0)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, p, p); err != nil {
				rt.Fatalf("an unset cap must not fail a finite walk: %v", err)
			}
		})
	})
}

// pubsubBox is a controllable publisher: it can refuse subscriptions or
// publishes (preconditions), fail a drain, or deliver a message zero, one or
// many times (the delivery-bound violations).
type pubsubBox struct {
	subErr, pubErr, drainErr error
	deliveries               int
}

func (p *pubsubBox) sub() (int, error) { return 0, p.subErr }
func (p *pubsubBox) publish() error    { return p.pubErr }
func (p *pubsubBox) drain(m string) ([]string, error) {
	if p.drainErr != nil {
		return nil, p.drainErr
	}
	out := make([]string, p.deliveries)
	for i := range out {
		out[i] = m
	}
	return out, nil
}

func TestPublisherDeliversBranches(t *testing.T) {
	t.Parallel()

	mk := func(b *pubsubBox, subscribers int) law.PublisherDelivers[*pubsubBox, string, int] {
		return law.PublisherDelivers[*pubsubBox, string, int]{
			Subscribe: func(_ *rapid.T, s *pubsubBox) (int, error) { return s.sub() },
			Publish:   func(_ *rapid.T, s *pubsubBox, _ string) error { return s.publish() },
			Drain: func(_ *rapid.T, s *pubsubBox, _ int) ([]string, error) {
				return s.drain("m")
			},
			Messages:    rapid.Just("m"),
			Subscribers: subscribers,
		}
	}

	t.Run("delivery to every subscriber passes", func(t *testing.T) {
		t.Parallel()
		b := &pubsubBox{deliveries: 1}
		l := mk(b, 3)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, b, &pubsubBox{deliveries: 1}); err != nil {
				rt.Fatalf("every subscriber received the message: %v", err)
			}
		})
	})

	t.Run("a refused subscribe or publish holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			noSub := &pubsubBox{subErr: errors.New("closed"), deliveries: 1}
			if err := mk(noSub, 2).Check(rt, noSub, noSub); !law.Holds(err) {
				rt.Fatalf("a refused subscribe is a precondition: %v", err)
			}
			noPub := &pubsubBox{pubErr: errors.New("closed"), deliveries: 1}
			if err := mk(noPub, 2).Check(rt, noPub, noPub); !law.Holds(err) {
				rt.Fatalf("a refused publish is a precondition: %v", err)
			}
		})
	})

	t.Run("a failing drain is a violation", func(t *testing.T) {
		t.Parallel()
		b := &pubsubBox{drainErr: errors.New("io"), deliveries: 1}
		l := mk(b, 2)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, b, &pubsubBox{deliveries: 1}); err == nil {
				rt.Fatal("a subscriber whose drain fails is a violation")
			}
		})
	})

	t.Run("a subscriber that never receives is a violation", func(t *testing.T) {
		t.Parallel()
		b := &pubsubBox{deliveries: 0}
		l := mk(b, 2)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, b, b); err == nil {
				rt.Fatal("a published message must reach every subscriber")
			}
		})
	})

	// An unset subscriber count fans out to a default rather than subscribing
	// nobody, which would make the law vacuous.
	t.Run("a non-positive subscriber count defaults", func(t *testing.T) {
		t.Parallel()
		subs := 0
		b := &pubsubBox{deliveries: 1}
		l := mk(b, 0)
		l.Subscribe = func(_ *rapid.T, s *pubsubBox) (int, error) { subs++; return s.sub() }
		rapid.Check(t, func(rt *rapid.T) {
			subs = 0
			_ = l.Check(rt, b, b)
			if subs < 2 {
				rt.Fatalf("the default fan-out must subscribe several handles, got %d", subs)
			}
		})
	})
}

// The three delivery modes bound the per-subscriber count differently, and the
// point of the law is that each rejects a different shape of misbehaviour.
func TestPublisherDeliveryBoundModes(t *testing.T) {
	t.Parallel()

	mk := func(b *pubsubBox, mode law.DeliveryMode) law.PublisherDeliveryBound[*pubsubBox, string, int] {
		return law.PublisherDeliveryBound[*pubsubBox, string, int]{
			Subscribe: func(_ *rapid.T, s *pubsubBox) (int, error) { return s.sub() },
			Publish:   func(_ *rapid.T, s *pubsubBox, _ string) error { return s.publish() },
			Drain: func(_ *rapid.T, s *pubsubBox, _ int) ([]string, error) {
				return s.drain("m")
			},
			Messages:    rapid.Just("m"),
			Subscribers: 2,
			Mode:        mode,
		}
	}
	check := func(t *testing.T, deliveries int, mode law.DeliveryMode) error {
		t.Helper()
		var got error
		rapid.Check(t, func(rt *rapid.T) {
			b := &pubsubBox{deliveries: deliveries}
			if err := mk(b, mode).Check(rt, b, b); got == nil {
				got = err
			}
		})
		return got
	}

	t.Run("at-least-once rejects a lost message", func(t *testing.T) {
		t.Parallel()
		if check(t, 1, law.DeliveryAtLeastOnce) != nil {
			t.Fatal("one delivery satisfies at-least-once")
		}
		if check(t, 2, law.DeliveryAtLeastOnce) != nil {
			t.Fatal("duplicates are permitted under at-least-once")
		}
		if check(t, 0, law.DeliveryAtLeastOnce) == nil {
			t.Fatal("a lost message violates at-least-once")
		}
	})

	t.Run("at-most-once rejects a duplicate", func(t *testing.T) {
		t.Parallel()
		if check(t, 0, law.DeliveryAtMostOnce) != nil {
			t.Fatal("loss is permitted under at-most-once")
		}
		if check(t, 1, law.DeliveryAtMostOnce) != nil {
			t.Fatal("one delivery satisfies at-most-once")
		}
		if check(t, 2, law.DeliveryAtMostOnce) == nil {
			t.Fatal("a duplicate violates at-most-once")
		}
	})

	t.Run("exactly-once rejects both loss and duplication", func(t *testing.T) {
		t.Parallel()
		if check(t, 1, law.DeliveryExactlyOnce) != nil {
			t.Fatal("one delivery satisfies exactly-once")
		}
		if check(t, 0, law.DeliveryExactlyOnce) == nil {
			t.Fatal("a lost message violates exactly-once")
		}
		if check(t, 2, law.DeliveryExactlyOnce) == nil {
			t.Fatal("a duplicate violates exactly-once")
		}
	})

	t.Run("a failing drain is a violation under any mode", func(t *testing.T) {
		t.Parallel()
		var got error
		rapid.Check(t, func(rt *rapid.T) {
			b := &pubsubBox{drainErr: errors.New("io")}
			if err := mk(b, law.DeliveryExactlyOnce).Check(rt, b, &pubsubBox{deliveries: 1}); got == nil {
				got = err
			}
		})
		if got == nil {
			t.Fatal("a drain failure is a violation regardless of mode")
		}
	})

	// Redeliver models a broker replaying a message; under at-most-once that
	// replay is exactly what the law must catch.
	t.Run("Redeliver runs when supplied", func(t *testing.T) {
		t.Parallel()
		redelivered := false
		rapid.Check(t, func(rt *rapid.T) {
			b := &pubsubBox{deliveries: 1}
			l := mk(b, law.DeliveryAtLeastOnce)
			l.Redeliver = func(*rapid.T, *pubsubBox, string) { redelivered = true }
			_ = l.Check(rt, b, b)
		})
		if !redelivered {
			t.Fatal("a supplied Redeliver hook must be invoked")
		}
	})

	t.Run("a refused subscribe holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			b := &pubsubBox{subErr: errors.New("no capacity")}
			if err := mk(b, law.DeliveryExactlyOnce).Check(rt, b, b); !law.Holds(err) {
				rt.Fatalf("a broker that refuses subscribers is a precondition: %v", err)
			}
		})
	})

	t.Run("a refused publish holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			b := &pubsubBox{pubErr: errors.New("topic closed")}
			if err := mk(b, law.DeliveryExactlyOnce).Check(rt, b, b); !law.Holds(err) {
				rt.Fatalf("a broker that refuses the publish is a precondition: %v", err)
			}
		})
	})

	// Subscribers ≤ 0 must not mean "check nobody", which would make every
	// delivery mode vacuously satisfied.
	t.Run("a non-positive subscriber count falls back to the default", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			b := &pubsubBox{deliveries: 0}
			l := mk(b, law.DeliveryAtLeastOnce)
			l.Subscribers = 0
			if err := l.Check(rt, b, b); err == nil {
				rt.Fatal("a lost message must still be caught with the default subscriber count")
			}
		})
	})
}

// A transaction's uncommitted writes must be invisible to readers outside it.
// The law compares an outside read before the transaction with one taken while
// it is open — presence and value both have to hold still.
func TestTransactionNoMidTxVisibilityBranches(t *testing.T) {
	t.Parallel()

	type txStore struct {
		committed map[string]string
		pending   map[string]string
		leakOnPut bool // configuration: does an uncommitted write become visible?
		leaking   bool // state: has TxPut run yet?
		beginErr  error
	}
	mk := func() law.TransactionNoMidTxVisibility[*txStore, int, string, string] {
		return law.TransactionNoMidTxVisibility[*txStore, int, string, string]{
			Begin:      func(_ *rapid.T, s *txStore) (int, error) { return 0, s.beginErr },
			TxPut:      func(_ *rapid.T, _ *txStore, _ int, k, v string) error { return nil },
			TxRollback: func(*rapid.T, *txStore, int) error { return nil },
			Read: func(_ *rapid.T, s *txStore, k string) (string, error) {
				if s.leaking {
					if v, ok := s.pending[k]; ok {
						return v, nil
					}
				}
				v, ok := s.committed[k]
				if !ok {
					return "", errors.New("not found")
				}
				return v, nil
			},
			Keys:   rapid.Just("k"),
			Values: rapid.Just("v"),
		}
	}
	fresh := func(leak bool) *txStore {
		return &txStore{
			committed: map[string]string{"k": "old"},
			pending:   map[string]string{"k": "new"},
			leakOnPut: leak,
		}
	}
	// withLeakAtPut wires TxPut to flip the store into its leaking state, so
	// the before/mid reads straddle the uncommitted write.
	withLeakAtPut := func(l law.TransactionNoMidTxVisibility[*txStore, int, string, string], s *txStore,
	) law.TransactionNoMidTxVisibility[*txStore, int, string, string] {
		l.TxPut = func(*rapid.T, *txStore, int, string, string) error {
			s.leaking = s.leakOnPut
			return nil
		}
		return l
	}

	t.Run("an isolated transaction passes", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := fresh(false)
			if err := withLeakAtPut(mk(), s).Check(rt, s, s); err != nil {
				rt.Fatalf("an isolated write is invisible outside: %v", err)
			}
		})
	})

	t.Run("a leaked uncommitted value is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := fresh(true)
			if err := withLeakAtPut(mk(), s).Check(rt, s, s); err == nil {
				rt.Fatal("an uncommitted write visible outside is a violation")
			}
		})
	})

	// A key that did not exist before but is readable mid-transaction is the
	// presence-level version of the same leak.
	t.Run("a key materialising mid-transaction is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := &txStore{
				committed: map[string]string{},
				pending:   map[string]string{"k": "new"},
				leakOnPut: true,
			}
			if err := withLeakAtPut(mk(), s).Check(rt, s, s); err == nil {
				rt.Fatal("a key appearing mid-transaction is a violation")
			}
		})
	})

	t.Run("a refused Begin or TxPut holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			s := fresh(true)
			s.beginErr = errors.New("no tx")
			if err := withLeakAtPut(mk(), s).Check(rt, s, s); !law.Holds(err) {
				rt.Fatalf("a refused Begin is a precondition: %v", err)
			}

			s2 := fresh(true)
			l := mk()
			l.TxPut = func(*rapid.T, *txStore, int, string, string) error { return errors.New("rejected") }
			if err := l.Check(rt, s2, s2); !law.Holds(err) {
				rt.Fatalf("a refused TxPut is a precondition: %v", err)
			}
		})
	})
}

// A lease taken under a context must free once that context is cancelled.
// The law polls until Timeout, so both the "never frees" and the "already free
// before cancel" defects have to be distinguishable.
func TestLeaseReleasedOnCancelBranches(t *testing.T) {
	t.Parallel()

	// held is atomic because the release runs on the context's own goroutine
	// while the law polls Free from the caller's — the same cross-goroutine
	// handoff a real lease has.
	type leaseBox struct {
		held       atomic.Bool
		acquireErr error
		neverFrees bool
	}
	mk := func(b *leaseBox, timeout time.Duration) law.LeaseReleasedOnCancel[*leaseBox, string] {
		return law.LeaseReleasedOnCancel[*leaseBox, string]{
			Acquire: func(ctx context.Context, s *leaseBox, _ string) error {
				if s.acquireErr != nil {
					return s.acquireErr
				}
				s.held.Store(true)
				if !s.neverFrees {
					context.AfterFunc(ctx, func() { s.held.Store(false) })
				}
				return nil
			},
			Free:    func(_ *rapid.T, s *leaseBox, _ string) bool { return !s.held.Load() },
			Keys:    rapid.Just("k"),
			Timeout: timeout,
		}
	}

	t.Run("a lease that frees on cancel passes", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			b := &leaseBox{}
			if err := mk(b, 2*time.Second).Check(rt, b, b); err != nil {
				rt.Fatalf("a lease released on cancel must pass: %v", err)
			}
		})
	})

	t.Run("a refused acquire holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			b := &leaseBox{acquireErr: errors.New("held elsewhere")}
			if err := mk(b, time.Second).Check(rt, b, b); !law.Holds(err) {
				rt.Fatalf("a lease that cannot be taken is a precondition: %v", err)
			}
		})
	})

	// Reporting free while the caller still holds it is a different defect
	// from never releasing, and the law distinguishes them.
	t.Run("a lease free immediately after acquire is a violation", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			b := &leaseBox{}
			l := mk(b, time.Second)
			l.Free = func(*rapid.T, *leaseBox, string) bool { return true }
			err := l.Check(rt, b, b)
			if err == nil {
				rt.Fatal("a lease reported free while held is a violation")
			}
			if !strings.Contains(err.Error(), "immediately after acquire") {
				rt.Fatalf("the diagnostic must name the defect, got: %v", err)
			}
		})
	})

	t.Run("a lease that never releases times out", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			b := &leaseBox{neverFrees: true}
			err := mk(b, 20*time.Millisecond).Check(rt, b, b)
			if err == nil {
				rt.Fatal("a lease that outlives its context is a violation")
			}
			if !strings.Contains(err.Error(), "not released") {
				rt.Fatalf("the diagnostic must name the timeout, got: %v", err)
			}
		})
	})

	// Timeout ≤ 0 must not mean "give up instantly" — a lease released a
	// moment after cancel is correct, and the default window allows for it.
	t.Run("a non-positive timeout falls back to the default", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			b := &leaseBox{}
			if err := mk(b, 0).Check(rt, b, b); err != nil {
				rt.Fatalf("a lease released on cancel must pass under the default window: %v", err)
			}
		})
	})
}

// The pair tests below hold every mirrored contract law to the conduct
// contract on [law.Law]: each mutation the subject accepts lands on the
// reference, and a refusal is the law's to report. The other subtests pass
// one store as both sides — the arrangement that cannot see a diverged pair.

func TestPersisterRetrievablePair(t *testing.T) {
	t.Parallel()

	t.Run("the accepted save lands on both sides", func(t *testing.T) {
		t.Parallel()
		l := law.PersisterRetrievable[*persisterSUT, string, int]{
			Save:   func(rt *rapid.T, s *persisterSUT, v string) (int, error) { return s.save(rt, v) },
			Read:   func(rt *rapid.T, s *persisterSUT, id int) (string, error) { return s.read(rt, id) },
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut, ref := &persisterSUT{}, &persisterSUT{}
			if err := l.Check(rt, sut, ref); err != nil {
				rt.Fatal(err)
			}
			if len(ref.store) == 0 {
				rt.Fatal("the reference never saw the save: the pair has diverged")
			}
		})
	})

	t.Run("a refusing reference is reported", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			ref := &persisterSUT{}
			l := law.PersisterRetrievable[*persisterSUT, string, int]{
				Save: func(rt *rapid.T, s *persisterSUT, v string) (int, error) {
					if s == ref {
						return 0, errors.New("refused")
					}
					return s.save(rt, v)
				},
				Read:   func(rt *rapid.T, s *persisterSUT, id int) (string, error) { return s.read(rt, id) },
				Values: rapid.Just("v"),
			}
			if err := l.Check(rt, &persisterSUT{}, ref); err == nil {
				rt.Fatal("expected the refusal to be reported")
			}
		})
	})
}

func TestUpdaterReplacesPair(t *testing.T) {
	t.Parallel()

	t.Run("both updates land on both sides", func(t *testing.T) {
		t.Parallel()
		l := law.UpdaterReplaces[*updaterSUT, string, string]{
			Update: func(rt *rapid.T, u *updaterSUT, v string) error { return u.write(rt, v) },
			Read:   func(rt *rapid.T, u *updaterSUT, k string) (string, error) { return u.lookup(rt, k) },
			Values: rapid.Just("aa"),
			KeyOf:  func(v string) string { return v[:1] },
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut, ref := &updaterSUT{}, &updaterSUT{}
			if err := l.Check(rt, sut, ref); err != nil {
				rt.Fatal(err)
			}
			if _, err := ref.lookup(rt, "a"); err != nil {
				rt.Fatal("the reference never saw the updates: the pair has diverged")
			}
		})
	})

	t.Run("a reference refusing either mirror is reported", func(t *testing.T) {
		t.Parallel()
		for refuseAt := 1; refuseAt <= 2; refuseAt++ {
			rapid.Check(t, func(rt *rapid.T) {
				ref, refCalls := &updaterSUT{}, 0
				l := law.UpdaterReplaces[*updaterSUT, string, string]{
					Update: func(rt *rapid.T, u *updaterSUT, v string) error {
						if u == ref {
							if refCalls++; refCalls == refuseAt {
								return errors.New("refused")
							}
						}
						return u.write(rt, v)
					},
					Read:   func(rt *rapid.T, u *updaterSUT, k string) (string, error) { return u.lookup(rt, k) },
					Values: rapid.Just("aa"),
					KeyOf:  func(v string) string { return v[:1] },
				}
				if err := l.Check(rt, &updaterSUT{}, ref); err == nil {
					rt.Fatalf("expected the reference's refusal #%d to be reported", refuseAt)
				}
			})
		}
	})
}

func TestUpserterIdempotentPair(t *testing.T) {
	t.Parallel()

	t.Run("both upserts land on both sides", func(t *testing.T) {
		t.Parallel()
		l := law.UpserterIdempotent[*updaterSUT, string, string]{
			Upsert: func(rt *rapid.T, u *updaterSUT, v string) error { return u.write(rt, v) },
			Read:   func(rt *rapid.T, u *updaterSUT, k string) (string, error) { return u.lookup(rt, k) },
			Values: rapid.Just("aa"),
			KeyOf:  func(v string) string { return v[:1] },
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut, ref := &updaterSUT{}, &updaterSUT{}
			if err := l.Check(rt, sut, ref); err != nil {
				rt.Fatal(err)
			}
			if _, err := ref.lookup(rt, "a"); err != nil {
				rt.Fatal("the reference never saw the upserts: the pair has diverged")
			}
		})
	})

	t.Run("a subject refusing its repeat is reported", func(t *testing.T) {
		t.Parallel()
		// The first upsert is the precondition; the second is the claim.
		rapid.Check(t, func(rt *rapid.T) {
			sut, sutCalls := &updaterSUT{}, 0
			l := law.UpserterIdempotent[*updaterSUT, string, string]{
				Upsert: func(rt *rapid.T, u *updaterSUT, v string) error {
					if u == sut {
						if sutCalls++; sutCalls == 2 {
							return errors.New("refused")
						}
					}
					return u.write(rt, v)
				},
				Read:   func(rt *rapid.T, u *updaterSUT, k string) (string, error) { return u.lookup(rt, k) },
				Values: rapid.Just("aa"),
				KeyOf:  func(v string) string { return v[:1] },
			}
			if err := l.Check(rt, sut, &updaterSUT{}); err == nil {
				rt.Fatal("expected the refused repeat to be reported")
			}
		})
	})

	t.Run("a reference refusing either mirror is reported", func(t *testing.T) {
		t.Parallel()
		for refuseAt := 1; refuseAt <= 2; refuseAt++ {
			rapid.Check(t, func(rt *rapid.T) {
				ref, refCalls := &updaterSUT{}, 0
				l := law.UpserterIdempotent[*updaterSUT, string, string]{
					Upsert: func(rt *rapid.T, u *updaterSUT, v string) error {
						if u == ref {
							if refCalls++; refCalls == refuseAt {
								return errors.New("refused")
							}
						}
						return u.write(rt, v)
					},
					Read:   func(rt *rapid.T, u *updaterSUT, k string) (string, error) { return u.lookup(rt, k) },
					Values: rapid.Just("aa"),
					KeyOf:  func(v string) string { return v[:1] },
				}
				if err := l.Check(rt, &updaterSUT{}, ref); err == nil {
					rt.Fatalf("expected the reference's refusal #%d to be reported", refuseAt)
				}
			})
		}
	})
}

func TestAppenderMonotonicOffsetsPair(t *testing.T) {
	t.Parallel()

	t.Run("the accepted append lands on both sides", func(t *testing.T) {
		t.Parallel()
		l := &law.AppenderMonotonicOffsets[*ref.MonotonicLog[string], string, int64]{
			Append: func(rt *rapid.T, s *ref.MonotonicLog[string], v string) (int64, error) {
				return s.Append(rt.Context(), v)
			},
			Values: rapid.Just("v"),
		}
		// The law remembers the last offset across iterations, so the pair
		// persists with it — a fresh log per iteration would restart offsets
		// under a memory that does not.
		sut, refLog := ref.NewMonotonicLog[string](), ref.NewMonotonicLog[string]()
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, sut, refLog); err != nil {
				rt.Fatal(err)
			}
			if refLog.Len() == 0 {
				rt.Fatal("the reference never saw the append: the pair has diverged")
			}
		})
	})

	t.Run("a refusing reference is reported", func(t *testing.T) {
		t.Parallel()
		refLog := ref.NewMonotonicLog[string]()
		sut := ref.NewMonotonicLog[string]()
		l := &law.AppenderMonotonicOffsets[*ref.MonotonicLog[string], string, int64]{
			Append: func(rt *rapid.T, s *ref.MonotonicLog[string], v string) (int64, error) {
				if s == refLog {
					return 0, errors.New("refused")
				}
				return s.Append(rt.Context(), v)
			},
			Values: rapid.Just("v"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, sut, refLog); err == nil {
				rt.Fatal("expected the refusal to be reported")
			}
		})
	})
}

func TestCASAtomicOneWinnerPair(t *testing.T) {
	t.Parallel()

	// The pair of attempts lands unasserted on the reference: on a
	// synchronized pair the cell's version arithmetic makes the outcomes
	// agree, so the landing signal is that the reference's attempt counter
	// moved at all.
	type cell struct{ v, attempts int }
	errStale := errors.New("stale")
	rapid.Check(t, func(rt *rapid.T) {
		sut, refCell := &cell{}, &cell{}
		l := law.CASAtomicOneWinner[*cell, int]{
			CAS: func(_ *rapid.T, c *cell, version int) error {
				c.attempts++
				if version != c.v {
					return errStale
				}
				c.v++
				return nil
			},
			Values:   rapid.Just(0),
			Mismatch: errStale,
		}
		if err := l.Check(rt, sut, refCell); err != nil {
			rt.Fatal(err)
		}
		if refCell.attempts != 2 {
			rt.Fatalf("the reference saw %d attempts, not the pair: the calls diverged", refCell.attempts)
		}
	})
}

func TestWatcherReturnsOnChangePair(t *testing.T) {
	t.Parallel()

	t.Run("the mutation lands on both sides", func(t *testing.T) {
		t.Parallel()
		l := law.WatcherReturnsOnChange[*watchable, chan int, string, int]{
			Watch:  func(_ *rapid.T, s *watchable, k string) (chan int, error) { return s.watch(k), nil },
			Mutate: func(_ *rapid.T, s *watchable, k string, v int) error { s.mutate(k, v); return nil },
			Next:   nextWatch,
			Stop:   func(_ chan int) {},
			Keys:   rapid.Just("k"),
			Values: rapid.Just(1),
		}
		refMutations := 0
		rapid.Check(t, func(rt *rapid.T) {
			refMutations = 0
			sut, refW := newWatchable(false), newWatchable(false)
			inner := l
			inner.Mutate = func(_ *rapid.T, s *watchable, k string, v int) error {
				if s == refW {
					refMutations++
				}
				s.mutate(k, v)
				return nil
			}
			if err := inner.Check(rt, sut, refW); err != nil {
				rt.Fatal(err)
			}
			if refMutations != 1 {
				rt.Fatal("the reference never saw the mutation: the pair has diverged")
			}
		})
	})

	t.Run("a refusing reference is reported", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			refW := newWatchable(false)
			l := law.WatcherReturnsOnChange[*watchable, chan int, string, int]{
				Watch: func(_ *rapid.T, s *watchable, k string) (chan int, error) { return s.watch(k), nil },
				Mutate: func(_ *rapid.T, s *watchable, k string, v int) error {
					if s == refW {
						return errors.New("refused")
					}
					s.mutate(k, v)
					return nil
				},
				Next:   nextWatch,
				Stop:   func(_ chan int) {},
				Keys:   rapid.Just("k"),
				Values: rapid.Just(1),
			}
			if err := l.Check(rt, newWatchable(false), refW); err == nil {
				rt.Fatal("expected the refusal to be reported")
			}
		})
	})
}

// TestPublisherPairRefusals holds the publisher mirrors' three refusal arms:
// a reference that cannot subscribe, publish or drain is reported by the law
// rather than left for the next action to misattribute.
func TestPublisherPairRefusals(t *testing.T) {
	t.Parallel()

	mkDelivers := func() law.PublisherDelivers[*pubsubBox, string, int] {
		return law.PublisherDelivers[*pubsubBox, string, int]{
			Subscribe:   func(_ *rapid.T, s *pubsubBox) (int, error) { return s.sub() },
			Publish:     func(_ *rapid.T, s *pubsubBox, _ string) error { return s.publish() },
			Drain:       func(_ *rapid.T, s *pubsubBox, _ int) ([]string, error) { return s.drain("m") },
			Messages:    rapid.Just("m"),
			Subscribers: 1,
		}
	}
	mkBound := func() law.PublisherDeliveryBound[*pubsubBox, string, int] {
		return law.PublisherDeliveryBound[*pubsubBox, string, int]{
			Subscribe: func(_ *rapid.T, s *pubsubBox) (int, error) { return s.sub() },
			Publish:   func(_ *rapid.T, s *pubsubBox, _ string) error { return s.publish() },
			Drain:     func(_ *rapid.T, s *pubsubBox, _ int) ([]string, error) { return s.drain("m") },
			Redeliver: func(_ *rapid.T, s *pubsubBox, _ string) {},
			Messages:  rapid.Just("m"),
			Mode:      law.DeliveryAtLeastOnce,
		}
	}

	refusals := []*pubsubBox{
		{subErr: errors.New("refused"), deliveries: 1},
		{pubErr: errors.New("refused"), deliveries: 1},
		{drainErr: errors.New("refused"), deliveries: 1},
	}
	for _, refBox := range refusals {
		l1, l2 := mkDelivers(), mkBound()
		rapid.Check(t, func(rt *rapid.T) {
			if err := l1.Check(rt, &pubsubBox{deliveries: 1}, refBox); err == nil {
				rt.Fatal("PublisherDelivers: expected the refusal to be reported")
			}
			if err := l2.Check(rt, &pubsubBox{deliveries: 1}, refBox); err == nil {
				rt.Fatal("PublisherDeliveryBound: expected the refusal to be reported")
			}
		})
	}
}
