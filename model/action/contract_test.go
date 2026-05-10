// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/action"
)

func TestPersister(t *testing.T) {
	t.Parallel()

	t.Run("matching IDs pass", func(t *testing.T) {
		t.Parallel()
		a := action.Persister("Save", rapid.Just("v"),
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
		a := action.Persister("Save", rapid.Just("v"),
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
		a := action.Updater("Update", rapid.Just("v"),
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
		a := action.Upserter("Upsert", rapid.Just("v"),
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
		a := action.CompareAndSwap("Put", rapid.Just("v"),
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
		a := action.Appender("Append", rapid.Just("v"),
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
		a := action.Appender("Append", rapid.Just("v"),
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
		a := action.Watcher("Watch",
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
		a := action.Watcher("Watch",
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
		a := action.Paginator("List", rapid.Just(0),
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

	t.Run("page-limit overflow flagged", func(t *testing.T) {
		t.Parallel()
		a := action.Paginator("List", rapid.Just(0),
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
		a := action.GetOrCompute("Compute", rapid.Just("k"),
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
		a := action.TransactionFunc("InTx",
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
		a := action.AcquireLease("Acquire",
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
		a := action.Publisher("Publish", rapid.Just("msg"),
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
		a := action.Subscriber("Sub",
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
