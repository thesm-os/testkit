// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
)

// countingLaw records how often it ran and answers with a fixed error.
type countingLaw struct {
	calls int
	err   error
}

func (*countingLaw) ID() string    { return "TEST-COUNTING" }
func (*countingLaw) REQID() string { return "REQ-TEST-001" }

func (l *countingLaw) Check(*rapid.T, int, int) error {
	l.calls++
	return l.err
}

func TestBudget(t *testing.T) {
	t.Parallel()

	t.Run("caps invocations at n and refills on reset", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			inner := &countingLaw{err: errors.New("would fail if run")}
			l := law.Budget[int](2, inner)
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("the first invocation must reach the inner law")
			}
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("the second invocation must reach the inner law")
			}
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("beyond the budget the law must pass, got %v", err)
			}
			if inner.calls != 2 {
				rt.Fatalf("inner ran %d times, want the budget's 2", inner.calls)
			}
			// The runner's per-iteration reset refills the budget — the
			// property that keeps rapid's shrink re-runs deterministic.
			l.(law.Resettable).Reset()
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("after the reset the budget must refill and reach the inner law")
			}
			if inner.calls != 3 {
				rt.Fatalf("inner ran %d times after the refill, want 3", inner.calls)
			}
		})
	})

	t.Run("forwards identity", func(t *testing.T) {
		t.Parallel()
		l := law.Budget[int](1, &countingLaw{})
		if l.ID() != "TEST-COUNTING" || l.REQID() != "REQ-TEST-001" {
			t.Errorf("the combinator must report the wrapped law's name, got %s/%s", l.ID(), l.REQID())
		}
	})
}

// producedLaw test double: judges the secondary it is handed.
type secondaryLaw struct {
	sawBoth bool
	got     []string
}

func (*secondaryLaw) ID() string    { return "TEST-SECONDARY" }
func (*secondaryLaw) REQID() string { return "" }

func (l *secondaryLaw) Check(_ *rapid.T, sut, ref string) error {
	l.sawBoth = sut == ref
	l.got = append(l.got, sut)
	return nil
}

func TestProduced(t *testing.T) {
	t.Parallel()

	t.Run("opens one fresh secondary per check", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			inner := &secondaryLaw{}
			n := 0
			l := law.Produced(func(*rapid.T, int) (string, error) {
				n++
				return string(rune('a' + n - 1)), nil
			}, inner)
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatal(err)
			}
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatal(err)
			}
			if len(inner.got) < 2 || inner.got[0] == inner.got[1] {
				rt.Fatalf("each check must open a fresh secondary, got %v", inner.got)
			}
			if !inner.sawBoth {
				rt.Fatal("the produced value must arrive as both sides of the pair")
			}
		})
	})

	t.Run("a refused open is vacuous, never a silent pass", func(t *testing.T) {
		t.Parallel()
		l := law.Produced(func(*rapid.T, int) (string, error) {
			return "", errors.New("refused")
		}, &secondaryLaw{})
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, 0, 0)
			if !errors.Is(err, law.Vacuous) {
				rt.Fatalf("a refused open must be Vacuous for the census, got %v", err)
			}
		})
	})

	t.Run("forwards identity", func(t *testing.T) {
		t.Parallel()
		l := law.Produced(func(*rapid.T, int) (string, error) { return "", nil }, &secondaryLaw{})
		if l.ID() != "TEST-SECONDARY" {
			t.Errorf("the combinator must report the wrapped law's name, got %s", l.ID())
		}
	})
}
