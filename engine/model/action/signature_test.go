// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/action"
)

func TestReaderNoError(t *testing.T) {
	t.Parallel()

	t.Run("passes when SUT and ref agree", func(t *testing.T) {
		t.Parallel()
		a := action.ReaderNoError(
			"Get", rapid.Just("k"),
			func(_ context.Context, _ *simpleStore, _ string) string { return "v" },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("catches value mismatch", func(t *testing.T) {
		t.Parallel()
		i := 0
		a := action.ReaderNoError(
			"Get", rapid.Just("k"),
			func(_ context.Context, _ *simpleStore, _ string) string {
				i++
				if i%2 == 1 {
					return "sut"
				}
				return "ref"
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected mismatch")
			}
		})
	})
}

func TestPointerReader(t *testing.T) {
	t.Parallel()

	t.Run("both nil → no diff", func(t *testing.T) {
		t.Parallel()
		a := action.PointerReader(
			"Get", rapid.Just("k"),
			func(_ context.Context, _ *simpleStore, _ string) *string { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("SUT nil, ref non-nil flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		v := "x"
		a := action.PointerReader(
			"Get", rapid.Just("k"),
			func(_ context.Context, _ *simpleStore, _ string) *string {
				i++
				if i%2 == 1 {
					return nil
				}
				return &v
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected nil-mismatch")
			}
		})
	})

	t.Run("both non-nil pointed-value mismatch flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		x, y := "sut", "ref"
		a := action.PointerReader(
			"Get", rapid.Just("k"),
			func(_ context.Context, _ *simpleStore, _ string) *string {
				i++
				if i%2 == 1 {
					return &x
				}
				return &y
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected value-mismatch")
			}
		})
	})
}

func TestMultiReader(t *testing.T) {
	t.Parallel()

	t.Run("both error agree", func(t *testing.T) {
		t.Parallel()
		a := action.MultiReader(
			"Get", rapid.Just("k"),
			func(_ context.Context, _ *simpleStore, _ string) (string, int, error) {
				return "", 0, errBroken
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("V1 mismatch flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		a := action.MultiReader(
			"Get", rapid.Just("k"),
			func(_ context.Context, _ *simpleStore, _ string) (string, int, error) {
				i++
				if i%2 == 1 {
					return "sut", 1, nil
				}
				return "ref", 1, nil
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected V1 mismatch")
			}
		})
	})

	t.Run("V2 mismatch flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		a := action.MultiReader(
			"Get", rapid.Just("k"),
			func(_ context.Context, _ *simpleStore, _ string) (string, int, error) {
				i++
				if i%2 == 1 {
					return "v", 1, nil
				}
				return "v", 2, nil
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected V2 mismatch")
			}
		})
	})
}

func TestBatchReader(t *testing.T) {
	t.Parallel()

	t.Run("agreeing batches pass", func(t *testing.T) {
		t.Parallel()
		a := action.BatchReader(
			"List", rapid.Just("k"),
			func(_ context.Context, _ *simpleStore, ks ...string) ([]string, error) {
				return ks, nil
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

func TestCompositeWriter(t *testing.T) {
	t.Parallel()

	t.Run("matching errors pass", func(t *testing.T) {
		t.Parallel()
		a := action.CompositeWriter(
			"Set", rapid.Just("k"), rapid.Just("v"),
			func(_ context.Context, _ *simpleStore, _, _ string) error { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("mismatched errors flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		a := action.CompositeWriter(
			"Set", rapid.Just("k"), rapid.Just("v"),
			func(_ context.Context, _ *simpleStore, _, _ string) error {
				i++
				if i%2 == 1 {
					return errBroken
				}
				return nil
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected err-mismatch")
			}
		})
	})
}

func TestMultiArgWriter(t *testing.T) {
	t.Parallel()

	t.Run("matching errors pass", func(t *testing.T) {
		t.Parallel()
		a := action.MultiArgWriter(
			"Op",
			rapid.Just([]any{"a", 1, true}),
			func(_ context.Context, _ *simpleStore, _ []any) error { return nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})
}

func TestMultiAggregator(t *testing.T) {
	t.Parallel()

	t.Run("agreeing values pass", func(t *testing.T) {
		t.Parallel()
		a := action.MultiAggregator(
			"Stats",
			func(_ context.Context, _ *simpleStore) (int, int, error) { return 1, 2, nil },
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("V1 mismatch flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		a := action.MultiAggregator(
			"Stats",
			func(_ context.Context, _ *simpleStore) (int, int, error) {
				i++
				if i%2 == 1 {
					return 1, 2, nil
				}
				return 99, 2, nil
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected V1 mismatch")
			}
		})
	})

	t.Run("error mismatch flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		a := action.MultiAggregator(
			"Stats",
			func(_ context.Context, _ *simpleStore) (int, int, error) {
				i++
				if i%2 == 1 {
					return 0, 0, errBroken
				}
				return 1, 2, nil
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected error mismatch")
			}
		})
	})
}

func TestStreamConsumer(t *testing.T) {
	t.Parallel()

	t.Run("consume reads stream and agrees", func(t *testing.T) {
		t.Parallel()
		a := action.StreamConsumer(
			"ReadAll",
			func() io.Reader { return strings.NewReader("hello") },
			func(_ context.Context, _ *simpleStore, r io.Reader) (string, error) {
				b, err := io.ReadAll(r)
				if err != nil {
					return "", fmt.Errorf("read: %w", err)
				}
				return string(b), nil
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
	})

	t.Run("error mismatch flagged", func(t *testing.T) {
		t.Parallel()
		i := 0
		a := action.StreamConsumer(
			"ReadAll",
			func() io.Reader { return strings.NewReader("hello") },
			func(_ context.Context, _ *simpleStore, r io.Reader) (string, error) {
				_, _ = io.ReadAll(r)
				i++
				if i%2 == 1 {
					return "", errors.New("nope")
				}
				return "ok", nil
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			r := a.Run(rt, &simpleStore{}, &simpleStore{})
			if r.Err == nil {
				rt.Fatal("expected error mismatch")
			}
		})
	})
}

func TestVoidLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("calls both SUT and ref", func(t *testing.T) {
		t.Parallel()
		var sutCalls, refCalls int
		a := action.VoidLifecycle(
			"Reset",
			func(_ context.Context, s *simpleStore) {
				if s == nil {
					return
				}
			},
		)
		_ = a
		// Re-bind to capture which arg is which by using identity counters.
		a = action.VoidLifecycle(
			"Reset",
			func(_ context.Context, s *simpleStore) {
				if s.getF == nil {
					sutCalls++
				} else {
					refCalls++
				}
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			sut := &simpleStore{}
			ref := &simpleStore{getF: func(_ string) (string, error) { return "", nil }}
			r := a.Run(rt, sut, ref)
			if r.Err != nil {
				rt.Fatalf("unexpected: %v", r.Err)
			}
		})
		if sutCalls == 0 || refCalls == 0 {
			t.Fatalf("expected both sides called: sut=%d ref=%d", sutCalls, refCalls)
		}
	})
}
