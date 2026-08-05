// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
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

// A roundtrip law's whole content is the comparison at the end: every step
// that errors is a precondition, because a conversion the subject refused says
// nothing about whether the conversion is lossless.
func TestRoundtripLawPreconditions(t *testing.T) {
	t.Parallel()

	boom := errors.New("unsupported")

	t.Run("Roundtrip holds vacuously when Forward is refused", func(t *testing.T) {
		t.Parallel()
		l := law.Roundtrip[int, string]{
			Forward: func(*rapid.T, int, string) (string, error) { return "", boom },
			Inverse: func(_ *rapid.T, _ int, s string) (string, error) { return s, nil },
			Values:  rapid.Just("x"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a refused Forward is a precondition: %v", err)
			}
		})
	})

	t.Run("Roundtrip holds vacuously when Inverse is refused", func(t *testing.T) {
		t.Parallel()
		l := law.Roundtrip[int, string]{
			Forward: func(_ *rapid.T, _ int, s string) (string, error) { return s, nil },
			Inverse: func(*rapid.T, int, string) (string, error) { return "", boom },
			Values:  rapid.Just("x"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a refused Inverse is a precondition: %v", err)
			}
		})
	})

	t.Run("Roundtrip flags a value that does not survive the trip", func(t *testing.T) {
		t.Parallel()
		l := law.Roundtrip[int, string]{
			Forward: func(_ *rapid.T, _ int, s string) (string, error) { return s + "!", nil },
			Inverse: func(_ *rapid.T, _ int, s string) (string, error) { return s, nil },
			Values:  rapid.Just("x"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("a value changed by the round trip is a violation")
			}
		})
	})

	// Lossy round trips only require the *second* forward pass to agree with
	// the first — the value itself may change, the encoding must stabilise.
	t.Run("LossyRoundtrip accepts loss but requires stability", func(t *testing.T) {
		t.Parallel()
		lower := law.LossyRoundtrip[int, string]{
			Forward: func(_ *rapid.T, _ int, s string) (string, error) { return strings.ToLower(s), nil },
			Inverse: func(_ *rapid.T, _ int, s string) (string, error) { return s, nil },
			Values:  rapid.Just("MiXeD"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := lower.Check(rt, 0, 0); err != nil {
				rt.Fatalf("lowercasing is lossy but stable: %v", err)
			}
		})

		unstable := law.LossyRoundtrip[int, string]{
			Forward: func(_ *rapid.T, _ int, s string) (string, error) { return s + "!", nil },
			Inverse: func(_ *rapid.T, _ int, s string) (string, error) { return s, nil },
			Values:  rapid.Just("x"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := unstable.Check(rt, 0, 0); err == nil {
				rt.Fatal("an encoding that keeps growing never stabilises")
			}
		})
	})

	t.Run("LossyRoundtrip preconditions", func(t *testing.T) {
		t.Parallel()
		base := law.LossyRoundtrip[int, string]{
			Forward: func(_ *rapid.T, _ int, s string) (string, error) { return s, nil },
			Inverse: func(_ *rapid.T, _ int, s string) (string, error) { return s, nil },
			Values:  rapid.Just("x"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			noForward := base
			noForward.Forward = func(*rapid.T, int, string) (string, error) { return "", boom }
			if err := noForward.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a refused Forward is a precondition: %v", err)
			}

			noInverse := base
			noInverse.Inverse = func(*rapid.T, int, string) (string, error) { return "", boom }
			if err := noInverse.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a refused Inverse is a precondition: %v", err)
			}

			// The second Forward can also fail, and that is still a
			// precondition rather than an instability.
			calls := 0
			secondFails := base
			secondFails.Forward = func(_ *rapid.T, _ int, s string) (string, error) {
				calls++
				if calls == 2 {
					return "", boom
				}
				return s, nil
			}
			if err := secondFails.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a refused re-Forward is a precondition: %v", err)
			}
		})
	})
}
