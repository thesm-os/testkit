// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/action"
)

func TestPersister(t *testing.T) {
	t.Parallel()

	t.Run("matching IDs pass", func(t *testing.T) {
		t.Parallel()
		a := action.Persister(
			"Save", rapid.Just("v"),
			func(_ context.Context, _ *simpleStore, _ string) (int, error) { return 7, nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("ID mismatch flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		a := action.Persister(
			"Save", rapid.Just("v"),
			func(_ context.Context, _ *simpleStore, _ string) (int, error) {
				i++
				return i, nil
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected ID mismatch")
			}
		})
	})
}

func TestUpdaterAndUpserter(t *testing.T) {
	t.Parallel()

	t.Run("updater wraps Writer pattern", func(t *testing.T) {
		t.Parallel()
		a := action.Updater(
			"Update", rapid.Just("v"),
			func(_ context.Context, _ *simpleStore, _ string) error { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("upserter wraps Writer pattern", func(t *testing.T) {
		t.Parallel()
		a := action.Upserter(
			"Upsert", rapid.Just("v"),
			func(_ context.Context, _ *simpleStore, _ string) error { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})
}

func TestCompareAndSwap(t *testing.T) {
	t.Parallel()

	t.Run("matching outcomes pass", func(t *testing.T) {
		t.Parallel()
		a := action.CompareAndSwap(
			"Put", rapid.Just("v"),
			func(_ context.Context, _ *simpleStore, _ string) error { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})
}

func TestAppender(t *testing.T) {
	t.Parallel()

	t.Run("matching offsets pass", func(t *testing.T) {
		t.Parallel()
		a := action.Appender(
			"Append", rapid.Just("v"),
			func(_ context.Context, _ *simpleStore, _ string) (int64, error) { return 1, nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("offset mismatch flagged", func(t *testing.T) {
		t.Parallel()
		i := int64(0)
		a := action.Appender(
			"Append", rapid.Just("v"),
			func(_ context.Context, _ *simpleStore, _ string) (int64, error) {
				i++
				return i, nil
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected offset mismatch")
			}
		})
	})
}

func TestWatcher(t *testing.T) {
	t.Parallel()

	t.Run("both return non-nil channels", func(t *testing.T) {
		t.Parallel()
		ch := make(chan string, 1)
		a := action.Watcher(
			"Watch",
			func(_ context.Context, _ *simpleStore) (<-chan string, error) { return ch, nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("nil-vs-non-nil channel flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		ch := make(chan string, 1)
		a := action.Watcher(
			"Watch",
			func(_ context.Context, _ *simpleStore) (<-chan string, error) {
				i++
				if i%2 == 1 {
					return nil, nil
				}
				return ch, nil
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected channel mismatch")
			}
		})
	})
}

func TestPaginator(t *testing.T) {
	t.Parallel()

	t.Run("drains until zero cursor", func(t *testing.T) {
		t.Parallel()
		pages := [][]string{{"a", "b"}, {"c"}, {}}
		i := 0
		a := action.Paginator(
			"List", rapid.Just(0),
			func(_ context.Context, _ *simpleStore, _ int) ([]string, int, error) {
				p := pages[i%len(pages)]
				i++
				if len(p) == 0 {
					return p, 0, nil
				}
				return p, 1, nil
			},
			10,
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("a non-positive limit falls back to the default ceiling", func(t *testing.T) {
		t.Parallel()
		a := action.Paginator(
			"List", rapid.Just(0),
			func(context.Context, *simpleStore, int) ([]string, int, error) {
				return []string{"x"}, 1, nil // never returns the zero cursor
			},
			0,
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("an endless paginator must still hit the default limit")
			}
			if !strings.Contains(r.Err.Error(), "100") {
				rt.Fatalf("the diagnostic must name the substituted ceiling, got: %v", r.Err)
			}
		})
	})

	t.Run("page-limit overflow flagged", func(t *testing.T) {
		t.Parallel()
		a := action.Paginator(
			"List", rapid.Just(0),
			func(_ context.Context, _ *simpleStore, _ int) ([]string, int, error) {
				return []string{"x"}, 1, nil // never returns zero cursor
			},
			3,
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected page-limit error")
			}
		})
	})
}

func TestGetOrCompute(t *testing.T) {
	t.Parallel()

	t.Run("matching values pass", func(t *testing.T) {
		t.Parallel()
		a := action.GetOrCompute(
			"Compute", rapid.Just("k"),
			func() string { return "v" },
			func(_ context.Context, _ *simpleStore, _ string, fn func() string) (string, error) {
				return fn(), nil
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})
}

func TestTransactionFunc(t *testing.T) {
	t.Parallel()

	t.Run("matching outcomes pass", func(t *testing.T) {
		t.Parallel()
		a := action.TransactionFunc(
			"InTx",
			func(_ struct{}) error { return nil },
			func(_ context.Context, _ *simpleStore, body func(struct{}) error) error {
				return body(struct{}{})
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})
}

func TestAcquireLeaseAndPublisherSubscriber(t *testing.T) {
	t.Parallel()

	t.Run("acquire wraps Lifecycle", func(t *testing.T) {
		t.Parallel()
		a := action.AcquireLease(
			"Acquire",
			func(_ context.Context, _ *simpleStore) error { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("publisher wraps Writer", func(t *testing.T) {
		t.Parallel()
		a := action.Publisher(
			"Publish", rapid.Just("msg"),
			func(_ context.Context, _ *simpleStore, _ string) error { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("subscriber wraps Watcher", func(t *testing.T) {
		t.Parallel()
		ch := make(chan string, 1)
		a := action.Subscriber(
			"Sub",
			func(_ context.Context, _ *simpleStore) (<-chan string, error) { return ch, nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})
}

// Every contract action compares subject against reference. These drive the
// divergence arms — the only reason the actions exist — plus the drain limits
// that turn a non-terminating paginator into a diagnostic instead of a hang.
func TestContractActionDivergence(t *testing.T) {
	t.Parallel()

	type impl struct {
		err error
		id  int
	}
	// The first non-nil result across rapid's iterations is the finding; the
	// fixtures are values so each Run sees a fresh copy.
	firstErr := func(t *testing.T, a model.Action[*impl], sut, ref impl) error {
		t.Helper()
		var got error
		rapid.Check(t, func(rt *rapid.T) {
			s, r := sut, ref
			if res := a.Run(rt, &s, &r); got == nil {
				got = res.Err
			}
		})
		return got
	}

	t.Run("Persister reports an error divergence", func(t *testing.T) {
		t.Parallel()
		a := action.Persister("Save", rapid.Just("v"),
			func(_ context.Context, i *impl, _ string) (int, error) { return i.id, i.err })
		if firstErr(t, a, impl{err: errors.New("full")}, impl{}) == nil {
			t.Fatal("one side refusing a save is a divergence")
		}
	})

	t.Run("Persister reports an id divergence", func(t *testing.T) {
		t.Parallel()
		a := action.Persister("Save", rapid.Just("v"),
			func(_ context.Context, i *impl, _ string) (int, error) { return i.id, nil })
		if firstErr(t, a, impl{id: 1}, impl{id: 2}) == nil {
			t.Fatal("differing ids for the same value is a divergence")
		}
	})

	t.Run("Appender reports an offset divergence", func(t *testing.T) {
		t.Parallel()
		a := action.Appender("Append", rapid.Just("v"),
			func(_ context.Context, i *impl, _ string) (int, error) { return i.id, i.err })
		if firstErr(t, a, impl{id: 1}, impl{id: 9}) == nil {
			t.Fatal("differing offsets is a divergence")
		}
		if firstErr(t, a, impl{err: errors.New("closed")}, impl{}) == nil {
			t.Fatal("one side refusing an append is a divergence")
		}
	})

	t.Run("Watcher reports an open divergence", func(t *testing.T) {
		t.Parallel()
		a := action.Watcher("Watch", func(_ context.Context, i *impl) (<-chan int, error) {
			if i.err != nil {
				return nil, i.err
			}
			ch := make(chan int)
			return ch, nil
		})
		if firstErr(t, a, impl{err: errors.New("closed")}, impl{}) == nil {
			t.Fatal("one side failing to open a watch is a divergence")
		}
	})

	// A watcher that succeeds but hands back a nil channel is broken in a way
	// the error check alone would not catch.
	t.Run("Watcher reports a nil-channel divergence", func(t *testing.T) {
		t.Parallel()
		a := action.Watcher("Watch", func(_ context.Context, i *impl) (<-chan int, error) {
			if i.id == 0 {
				return nil, nil
			}
			ch := make(chan int)
			return ch, nil
		})
		if firstErr(t, a, impl{id: 0}, impl{id: 1}) == nil {
			t.Fatal("a nil channel opposite a real one is a divergence")
		}
	})

	t.Run("Paginator reports a non-terminating walk", func(t *testing.T) {
		t.Parallel()
		a := action.Paginator("Page", rapid.Just(1),
			func(_ context.Context, _ *impl, c int) ([]int, int, error) {
				return []int{c}, c + 1, nil // cursor never returns to zero
			}, 3)
		err := firstErr(t, a, impl{}, impl{})
		if err == nil || !strings.Contains(err.Error(), "limit") {
			t.Fatalf("a walk that exceeds the page limit must be reported, got: %v", err)
		}
	})

	t.Run("Paginator reports differing pages", func(t *testing.T) {
		t.Parallel()
		a := action.Paginator("Page", rapid.Just(1),
			func(_ context.Context, i *impl, _ int) ([]int, int, error) {
				return []int{i.id}, 0, nil // zero cursor ends the walk
			}, 10)
		if firstErr(t, a, impl{id: 1}, impl{id: 2}) == nil {
			t.Fatal("differing page contents is a divergence")
		}
	})

	t.Run("Paginator reports a mid-drain error", func(t *testing.T) {
		t.Parallel()
		a := action.Paginator("Page", rapid.Just(1),
			func(_ context.Context, i *impl, _ int) ([]int, int, error) {
				return nil, 0, i.err
			}, 10)
		if firstErr(t, a, impl{err: errors.New("io")}, impl{}) == nil {
			t.Fatal("a drain error is a fault regardless of agreement")
		}
	})

	t.Run("GetOrCompute reports error and value divergence", func(t *testing.T) {
		t.Parallel()
		a := action.GetOrCompute("Get", rapid.Just("k"),
			func() int { return 7 },
			func(_ context.Context, i *impl, _ string, c func() int) (int, error) {
				if i.err != nil {
					return 0, i.err
				}
				return c() + i.id, nil
			})
		if firstErr(t, a, impl{err: errors.New("miss")}, impl{}) == nil {
			t.Fatal("one side failing is a divergence")
		}
		if firstErr(t, a, impl{id: 1}, impl{id: 2}) == nil {
			t.Fatal("differing computed values is a divergence")
		}
	})

	t.Run("TransactionFunc reports an error divergence", func(t *testing.T) {
		t.Parallel()
		a := action.TransactionFunc("Tx",
			func(int) error { return nil },
			func(_ context.Context, i *impl, body func(int) error) error {
				if i.err != nil {
					return i.err
				}
				return body(0)
			})
		if firstErr(t, a, impl{err: errors.New("conflict")}, impl{}) == nil {
			t.Fatal("one side refusing the transaction is a divergence")
		}
	})
}
