// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

func TestRoundtrip(t *testing.T) {
	t.Parallel()

	t.Run("Inverse(Forward(x)) == x for negation", func(t *testing.T) {
		t.Parallel()
		l := law.Roundtrip[any, int]{
			Forward: func(_ *rapid.T, _ any, x int) (int, error) { return -x, nil },
			Inverse: func(_ *rapid.T, _ any, x int) (int, error) { return -x, nil },
			Values:  rapid.IntRange(-100, 100),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("non-roundtripping pair flagged", func(t *testing.T) {
		t.Parallel()
		l := law.Roundtrip[any, int]{
			Forward: func(_ *rapid.T, _ any, x int) (int, error) { return x + 1, nil },
			Inverse: func(_ *rapid.T, _ any, x int) (int, error) { return x + 1, nil }, // wrong inverse
			Values:  rapid.IntRange(1, 100),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err == nil {
				rt.Fatal("expected mismatch")
			}
		})
	})

	t.Run("errored forward is vacuous", func(t *testing.T) {
		t.Parallel()
		l := law.Roundtrip[any, int]{
			Forward: func(_ *rapid.T, _ any, _ int) (int, error) { return 0, errors.New("nope") },
			Inverse: func(_ *rapid.T, _ any, x int) (int, error) { return x, nil },
			Values:  rapid.Just(1),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestLossyRoundtrip(t *testing.T) {
	t.Parallel()

	t.Run("idempotent quantization passes", func(t *testing.T) {
		t.Parallel()
		// f rounds down to multiples of 10; inverse is identity. f(f(x)) == f(x).
		f := func(x int) int { return (x / 10) * 10 }
		l := law.LossyRoundtrip[any, int]{
			Forward: func(_ *rapid.T, _ any, x int) (int, error) { return f(x), nil },
			Inverse: func(_ *rapid.T, _ any, x int) (int, error) { return x, nil },
			Values:  rapid.IntRange(0, 1000),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("non-idempotent forward flagged", func(t *testing.T) {
		t.Parallel()
		l := law.LossyRoundtrip[any, int]{
			Forward: func(_ *rapid.T, _ any, x int) (int, error) { return x + 1, nil },
			Inverse: func(_ *rapid.T, _ any, x int) (int, error) { return x, nil },
			Values:  rapid.IntRange(1, 100),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err == nil {
				rt.Fatal("expected mismatch")
			}
		})
	})
}

func TestTotalOver(t *testing.T) {
	t.Parallel()

	t.Run("function defined over entire domain passes", func(t *testing.T) {
		t.Parallel()
		l := law.TotalOver[any, int, int]{
			Call:  func(_ *rapid.T, _ any, x int) (int, error) { return x * 2, nil },
			Input: rapid.IntRange(-100, 100),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("erroring on any input flagged", func(t *testing.T) {
		t.Parallel()
		l := law.TotalOver[any, int, int]{
			Call:  func(_ *rapid.T, _ any, _ int) (int, error) { return 0, errors.New("not total") },
			Input: rapid.Just(0),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err == nil {
				rt.Fatal("expected totality violation")
			}
		})
	})
}
