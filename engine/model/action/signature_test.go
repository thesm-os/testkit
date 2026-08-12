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

	"go.thesmos.sh/testkit/engine/model"
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

// Every signature action is a differential comparison, so the arm that matters
// is the one where the subject and the reference disagree on whether the call
// succeeded at all. Agreement — including agreeing to fail — is not a finding.
func TestSignatureActionErrorDivergence(t *testing.T) {
	t.Parallel()

	type sig struct{ err error }
	firstErr := func(t *testing.T, a model.Action[*sig], sut, ref sig) error {
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
	boom := errors.New("unavailable")

	t.Run("MultiReader", func(t *testing.T) {
		t.Parallel()
		a := action.MultiReader("Get", rapid.Just("k"),
			func(_ context.Context, s *sig, _ string) (int, int, error) { return 0, 0, s.err })
		if firstErr(t, a, sig{err: boom}, sig{}) == nil {
			t.Fatal("one side failing is a divergence")
		}
		if firstErr(t, a, sig{err: boom}, sig{err: boom}) != nil {
			t.Fatal("both sides failing is not a divergence")
		}
	})

	t.Run("BatchReader", func(t *testing.T) {
		t.Parallel()
		a := action.BatchReader("GetAll", rapid.Just("k"),
			func(_ context.Context, s *sig, _ ...string) ([]int, error) { return nil, s.err })
		if firstErr(t, a, sig{err: boom}, sig{}) == nil {
			t.Fatal("one side failing is a divergence")
		}
	})

	t.Run("MultiArgWriter", func(t *testing.T) {
		t.Parallel()
		a := action.MultiArgWriter("Put", rapid.Just([]any{1, "x"}),
			func(_ context.Context, s *sig, _ []any) error { return s.err })
		if firstErr(t, a, sig{err: boom}, sig{}) == nil {
			t.Fatal("one side failing is a divergence")
		}
	})

	t.Run("MultiAggregator", func(t *testing.T) {
		t.Parallel()
		a := action.MultiAggregator("Stats",
			func(_ context.Context, s *sig) (int, int, error) { return 0, 0, s.err })
		if firstErr(t, a, sig{err: boom}, sig{}) == nil {
			t.Fatal("one side failing is a divergence")
		}
	})

	t.Run("StreamConsumer", func(t *testing.T) {
		t.Parallel()
		a := action.StreamConsumer("Consume",
			func() int { return 1 },
			func(_ context.Context, s *sig, _ int) (int, error) { return 0, s.err })
		if firstErr(t, a, sig{err: boom}, sig{}) == nil {
			t.Fatal("one side failing is a divergence")
		}
	})
}

// A multi-value reader must be compared on every return value: agreeing on the
// first and diverging on the second is still a divergence, and a comparison
// that stopped at V1 would miss it.
func TestMultiValueActionsCompareEveryReturn(t *testing.T) {
	t.Parallel()

	type pair struct{ a, b int }
	firstErr := func(t *testing.T, act model.Action[*pair], sut, ref pair) error {
		t.Helper()
		var got error
		rapid.Check(t, func(rt *rapid.T) {
			s, r := sut, ref
			if res := act.Run(rt, &s, &r); got == nil {
				got = res.Err
			}
		})
		return got
	}

	t.Run("MultiReader flags a second-value divergence", func(t *testing.T) {
		t.Parallel()
		act := action.MultiReader("Get", rapid.Just("k"),
			func(_ context.Context, p *pair, _ string) (int, int, error) { return p.a, p.b, nil })
		err := firstErr(t, act, pair{a: 1, b: 2}, pair{a: 1, b: 9})
		if err == nil || !strings.Contains(err.Error(), "V2") {
			t.Fatalf("a divergence in the second value must be reported, got: %v", err)
		}
	})

	t.Run("MultiAggregator flags a second-value divergence", func(t *testing.T) {
		t.Parallel()
		act := action.MultiAggregator("Stats",
			func(_ context.Context, p *pair) (int, int, error) { return p.a, p.b, nil })
		err := firstErr(t, act, pair{a: 1, b: 2}, pair{a: 1, b: 9})
		if err == nil || !strings.Contains(err.Error(), "V2") {
			t.Fatalf("a divergence in the second value must be reported, got: %v", err)
		}
	})

	// Batch and stream shapes carry a slice-valued result, where a
	// same-length-different-contents divergence is the easiest one to miss.
	t.Run("BatchReader flags a payload divergence", func(t *testing.T) {
		t.Parallel()
		act := action.BatchReader("GetAll", rapid.Just("k"),
			func(_ context.Context, p *pair, keys ...string) ([]int, error) {
				return []int{p.a, p.b * len(keys)}, nil
			})
		err := firstErr(t, act, pair{a: 1, b: 2}, pair{a: 1, b: 9})
		if err == nil || !strings.Contains(err.Error(), "disagree") {
			t.Fatalf("a divergent batch payload must be reported, got: %v", err)
		}
	})

	t.Run("StreamConsumer flags a payload divergence", func(t *testing.T) {
		t.Parallel()
		act := action.StreamConsumer("Consume",
			func() int { return 3 },
			func(_ context.Context, p *pair, in int) (int, error) { return p.b * in, nil })
		err := firstErr(t, act, pair{a: 1, b: 2}, pair{a: 1, b: 9})
		if err == nil || !strings.Contains(err.Error(), "disagree") {
			t.Fatalf("a divergent stream result must be reported, got: %v", err)
		}
	})
}

// TestPureVar pins the drawn-input pure comparison: agreement passes, a
// side-dependent answer is flagged with the drawn arguments in the message.
func TestPureVar(t *testing.T) {
	t.Parallel()

	args := rapid.Just([]any{"in"})
	agree := action.PureVar("Echo", args, func(s string, a []any) any {
		return s + a[0].(string)
	})
	rapid.Check(t, func(rt *rapid.T) {
		if res := agree.Run(rt, "x", "x"); res.Err != nil {
			t.Fatalf("identical sides must agree: %v", res.Err)
		}
		if res := agree.Run(rt, "x", "y"); res.Err == nil {
			t.Fatal("differing sides must be flagged")
		}
	})
}

// TestPredicateVar pins the drawn-input predicate comparison.
func TestPredicateVar(t *testing.T) {
	t.Parallel()

	args := rapid.Just([]any{2})
	agree := action.PredicateVar("Even", args, func(s int, a []any) bool {
		return (s+a[0].(int))%2 == 0
	})
	rapid.Check(t, func(rt *rapid.T) {
		if res := agree.Run(rt, 0, 2); res.Err != nil {
			t.Fatalf("sides agreeing about parity must pass: %v", res.Err)
		}
		if res := agree.Run(rt, 0, 1); res.Err == nil {
			t.Fatal("sides disagreeing about parity must be flagged")
		}
	})
}
