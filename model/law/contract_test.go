// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
	"go.thesmos.sh/testkit/model/refappender"
	"go.thesmos.sh/testkit/model/refcas"
	"go.thesmos.sh/testkit/model/refcursor"
	"go.thesmos.sh/testkit/model/reflease"
	"go.thesmos.sh/testkit/model/refpaginator"
	"go.thesmos.sh/testkit/model/refpool"
	"go.thesmos.sh/testkit/model/refpubsub"
	"go.thesmos.sh/testkit/model/reftxn"
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

func TestPaginatorNoDuplicates(t *testing.T) {
	t.Parallel()

	t.Run("CursorTable walk emits each element exactly once", func(t *testing.T) {
		t.Parallel()
		tab := refpaginator.NewCursorTable[int, int](func(a, b int) bool { return a < b })
		for i := range 10 {
			_ = tab.Put(t.Context(), i, i)
		}
		l := law.PaginatorNoDuplicates[*refpaginator.CursorTable[int, int], int, int, int]{
			Page: func(rt *rapid.T, s *refpaginator.CursorTable[int, int], cur int) ([]int, int, bool) {
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
		tab := refpaginator.NewCursorTable[int, int](func(a, b int) bool { return a < b })
		for i := range 10 {
			_ = tab.Put(t.Context(), i, i)
		}
		l := law.PaginatorResumable[*refpaginator.CursorTable[int, int], int, int]{
			Page: func(rt *rapid.T, s *refpaginator.CursorTable[int, int], cur int) ([]int, int, bool) {
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
}

func TestPublisherDelivers(t *testing.T) {
	t.Parallel()

	t.Run("at-least-once broker reaches every subscriber", func(t *testing.T) {
		t.Parallel()
		l := law.PublisherDelivers[*refpubsub.AtLeastOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *refpubsub.AtLeastOnce[int]) (int, error) {
				return s.Subscribe(rt.Context())
			},
			Publish: func(rt *rapid.T, s *refpubsub.AtLeastOnce[int], m int) error {
				return s.Publish(rt.Context(), m)
			},
			Drain: func(rt *rapid.T, s *refpubsub.AtLeastOnce[int], sub int) ([]int, error) {
				return s.Drain(rt.Context(), sub)
			},
			Messages:    rapid.Int(),
			Subscribers: 3,
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := refpubsub.NewAtLeastOnce[int]()
			if err := l.Check(rt, b, b); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("broker that drops everything is caught", func(t *testing.T) {
		t.Parallel()
		l := law.PublisherDelivers[*refpubsub.AtMostOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *refpubsub.AtMostOnce[int]) (int, error) {
				return s.Subscribe(rt.Context())
			},
			Publish: func(rt *rapid.T, s *refpubsub.AtMostOnce[int], m int) error {
				return s.Publish(rt.Context(), m)
			},
			Drain: func(rt *rapid.T, s *refpubsub.AtMostOnce[int], sub int) ([]int, error) {
				msgs, _, err := s.Drain(rt.Context(), sub)
				return msgs, err
			},
			Messages:    rapid.Int(),
			Subscribers: 3,
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := refpubsub.NewAtMostOnce[int](0) // capacity 0 → drops all
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
		l := law.PublisherDeliveryBound[*refpubsub.AtLeastOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *refpubsub.AtLeastOnce[int]) (int, error) { return s.Subscribe(rt.Context()) },
			Publish:   func(rt *rapid.T, s *refpubsub.AtLeastOnce[int], m int) error { return s.Publish(rt.Context(), m) },
			Redeliver: func(rt *rapid.T, s *refpubsub.AtLeastOnce[int], m int) { _ = s.Publish(rt.Context(), m) },
			Drain: func(rt *rapid.T, s *refpubsub.AtLeastOnce[int], sub int) ([]int, error) {
				return s.Drain(rt.Context(), sub)
			},
			Messages: rapid.Int(),
			Mode:     law.DeliveryAtLeastOnce,
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := refpubsub.NewAtLeastOnce[int]()
			if err := l.Check(rt, b, b); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("at-most-once: single publish counts <= 1", func(t *testing.T) {
		t.Parallel()
		l := law.PublisherDeliveryBound[*refpubsub.AtMostOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *refpubsub.AtMostOnce[int]) (int, error) { return s.Subscribe(rt.Context()) },
			Publish:   func(rt *rapid.T, s *refpubsub.AtMostOnce[int], m int) error { return s.Publish(rt.Context(), m) },
			Drain: func(rt *rapid.T, s *refpubsub.AtMostOnce[int], sub int) ([]int, error) {
				msgs, _, err := s.Drain(rt.Context(), sub)
				return msgs, err
			},
			Messages: rapid.Int(),
			Mode:     law.DeliveryAtMostOnce,
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := refpubsub.NewAtMostOnce[int](4)
			if err := l.Check(rt, b, b); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("exactly-once: replay of same id counts == 1", func(t *testing.T) {
		t.Parallel()
		var lastID int64
		l := law.PublisherDeliveryBound[*refpubsub.ExactlyOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *refpubsub.ExactlyOnce[int]) (int, error) { return s.Subscribe(rt.Context()) },
			Publish: func(rt *rapid.T, s *refpubsub.ExactlyOnce[int], m int) error {
				id, err := s.Publish(rt.Context(), m)
				lastID = id
				return err
			},
			Redeliver: func(rt *rapid.T, s *refpubsub.ExactlyOnce[int], m int) { _ = s.Replay(rt.Context(), lastID, m) },
			Drain: func(rt *rapid.T, s *refpubsub.ExactlyOnce[int], sub int) ([]int, error) {
				return s.Drain(rt.Context(), sub)
			},
			Messages: rapid.Int(),
			Mode:     law.DeliveryExactlyOnce,
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := refpubsub.NewExactlyOnce[int]()
			if err := l.Check(rt, b, b); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("exactly-once mode catches a broker that duplicates", func(t *testing.T) {
		t.Parallel()
		l := law.PublisherDeliveryBound[*refpubsub.AtLeastOnce[int], int, int]{
			Subscribe: func(rt *rapid.T, s *refpubsub.AtLeastOnce[int]) (int, error) { return s.Subscribe(rt.Context()) },
			Publish:   func(rt *rapid.T, s *refpubsub.AtLeastOnce[int], m int) error { return s.Publish(rt.Context(), m) },
			Redeliver: func(rt *rapid.T, s *refpubsub.AtLeastOnce[int], m int) { _ = s.Publish(rt.Context(), m) }, // duplicates
			Drain: func(rt *rapid.T, s *refpubsub.AtLeastOnce[int], sub int) ([]int, error) {
				return s.Drain(rt.Context(), sub)
			},
			Messages: rapid.Int(),
			Mode:     law.DeliveryExactlyOnce, // but broker duplicates → must fire
		}
		rapid.Check(t, func(rt *rapid.T) {
			b := refpubsub.NewAtLeastOnce[int]()
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
		l := law.TransactionNoMidTxVisibility[*reftxn.SnapshotIsolation[string, int], *reftxn.Tx[string, int], string, int]{
			Begin: func(rt *rapid.T, s *reftxn.SnapshotIsolation[string, int]) (*reftxn.Tx[string, int], error) {
				return s.Begin(rt.Context())
			},
			TxPut: func(rt *rapid.T, tx *reftxn.Tx[string, int], k string, v int) error {
				return tx.Put(rt.Context(), k, v)
			},
			TxRollback: func(rt *rapid.T, tx *reftxn.Tx[string, int]) error { return tx.Rollback(rt.Context()) },
			Read: func(rt *rapid.T, s *reftxn.SnapshotIsolation[string, int], k string) (int, error) {
				return s.Get(rt.Context(), k)
			},
			Keys:   rapid.SampledFrom([]string{"a", "b"}),
			Values: rapid.Int(),
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := reftxn.NewSnapshotIsolation[string, int](errNF)
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("store that leaks uncommitted writes is caught", func(t *testing.T) {
		t.Parallel()
		errNF := errors.New("not found")
		l := law.TransactionNoMidTxVisibility[*leakyTxStore, *leakyTx, string, int]{
			Begin:      func(_ *rapid.T, s *leakyTxStore) (*leakyTx, error) { return &leakyTx{store: s}, nil },
			TxPut:      func(_ *rapid.T, tx *leakyTx, k string, v int) error { tx.store.data[k] = v; return nil }, // BUG: writes through
			TxRollback: func(_ *rapid.T, _ *leakyTx) error { return nil },
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
		l := law.TwoPhaseCommitOrRollback[*reftxn.SnapshotIsolation[string, int], *reftxn.Tx[string, int]]{
			Begin: func(rt *rapid.T, s *reftxn.SnapshotIsolation[string, int]) (*reftxn.Tx[string, int], error) {
				return s.Begin(rt.Context())
			},
			Commit: func(rt *rapid.T, _ *reftxn.SnapshotIsolation[string, int], tx *reftxn.Tx[string, int]) error {
				return tx.Commit(rt.Context())
			},
			Rollback: func(rt *rapid.T, _ *reftxn.SnapshotIsolation[string, int], tx *reftxn.Tx[string, int]) error {
				return tx.Rollback(rt.Context())
			},
			Closed: reftxn.ErrTxClosed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			s := reftxn.NewSnapshotIsolation[string, int](errors.New("nf"))
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
}
