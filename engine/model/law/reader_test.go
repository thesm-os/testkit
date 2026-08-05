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

type cacheableSUT struct {
	values map[string]int
}

func TestCacheable(t *testing.T) {
	t.Parallel()

	t.Run("repeated read returns same value", func(t *testing.T) {
		t.Parallel()
		s := &cacheableSUT{values: map[string]int{"k": 42}}
		l := law.Cacheable[*cacheableSUT, string, int]{
			Keys: rapid.Just("k"),
			Read: func(_ *rapid.T, c *cacheableSUT, k string) (int, error) {
				return c.values[k], nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("flips between calls catches mismatch", func(t *testing.T) {
		t.Parallel()
		toggle := false
		l := law.Cacheable[*cacheableSUT, string, int]{
			Keys: rapid.Just("k"),
			Read: func(_ *rapid.T, _ *cacheableSUT, _ string) (int, error) {
				toggle = !toggle
				if toggle {
					return 1, nil
				}
				return 2, nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &cacheableSUT{}, &cacheableSUT{}); err == nil {
				rt.Fatal("expected mismatch")
			}
		})
	})
}

func TestDefaultOnError(t *testing.T) {
	t.Parallel()

	t.Run("error-coupled default passes", func(t *testing.T) {
		t.Parallel()
		l := law.DefaultOnError[*cacheableSUT, string, int]{
			Keys: rapid.Just("k"),
			Read: func(_ *rapid.T, _ *cacheableSUT, _ string) (int, error) {
				return 0, errors.New("boom")
			},
			Default: 0,
			Eq:      func(a, b int) bool { return a == b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &cacheableSUT{}, &cacheableSUT{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("non-default-on-error flagged", func(t *testing.T) {
		t.Parallel()
		l := law.DefaultOnError[*cacheableSUT, string, int]{
			Keys: rapid.Just("k"),
			Read: func(_ *rapid.T, _ *cacheableSUT, _ string) (int, error) {
				return 99, errors.New("boom")
			},
			Default: 0,
			Eq:      func(a, b int) bool { return a == b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &cacheableSUT{}, &cacheableSUT{}); err == nil {
				rt.Fatal("expected mismatch")
			}
		})
	})
}

func TestSticky(t *testing.T) {
	t.Parallel()

	t.Run("first observation persists across calls", func(t *testing.T) {
		t.Parallel()
		l := &law.Sticky[*cacheableSUT, string, int]{
			Keys: rapid.Just("k"),
			Read: func(_ *rapid.T, _ *cacheableSUT, _ string) (int, error) {
				return 42, nil
			},
			Eq: func(a, b int) bool { return a == b },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &cacheableSUT{}, &cacheableSUT{}); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

// PointInTime asks whether two consecutive reads of the same key agree, with
// an optional disturbance between them. The interesting cases are the ones
// where the two reads disagree about whether the key exists at all.
func TestPointInTimeBranches(t *testing.T) {
	t.Parallel()

	t.Run("stable reads pass", func(t *testing.T) {
		t.Parallel()
		l := law.PointInTime[int, string, int]{
			Read: func(*rapid.T, int, string) (int, error) { return 7, nil },
			Keys: rapid.Just("k"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("two identical reads must pass: %v", err)
			}
		})
	})

	t.Run("a key that appears between reads is flagged", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			reads := 0
			l := law.PointInTime[int, string, int]{
				Read: func(*rapid.T, int, string) (int, error) {
					reads++
					if reads == 1 {
						return 0, errors.New("not found")
					}
					return 7, nil
				},
				Keys: rapid.Just("k"),
			}
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("a key that materialises between reads is not point-in-time stable")
			}
		})
	})

	t.Run("a key that vanishes between reads is flagged", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			reads := 0
			l := law.PointInTime[int, string, int]{
				Read: func(*rapid.T, int, string) (int, error) {
					reads++
					if reads == 1 {
						return 7, nil
					}
					return 0, errors.New("gone")
				},
				Keys: rapid.Just("k"),
			}
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("a key that disappears between reads is not point-in-time stable")
			}
		})
	})

	// Two reads that both fail agree with each other, so the law has nothing
	// to compare and holds vacuously.
	t.Run("two failed reads hold vacuously", func(t *testing.T) {
		t.Parallel()
		l := law.PointInTime[int, string, int]{
			Read: func(*rapid.T, int, string) (int, error) { return 0, errors.New("absent") },
			Keys: rapid.Just("k"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a consistently absent key is a precondition: %v", err)
			}
		})
	})

	t.Run("Disturb runs between the two reads", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			disturbed := false
			l := law.PointInTime[int, string, int]{
				Read:    func(*rapid.T, int, string) (int, error) { return 1, nil },
				Disturb: func(*rapid.T, int, string) { disturbed = true },
				Keys:    rapid.Just("k"),
			}
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a store unaffected by the disturbance must pass: %v", err)
			}
			if !disturbed {
				rt.Fatal("Disturb must be invoked between the reads")
			}
		})
	})

	// Both reads succeed and disagree — the snapshot moved under the reader,
	// which is exactly what point-in-time reads forbid.
	t.Run("a value that changes between reads is flagged", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			reads := 0
			l := law.PointInTime[int, string, int]{
				Read: func(*rapid.T, int, string) (int, error) {
					reads++
					return reads, nil
				},
				Keys: rapid.Just("k"),
			}
			err := l.Check(rt, 0, 0)
			if err == nil || !strings.Contains(err.Error(), "snapshot drifted") {
				rt.Fatalf("a drifting value must be reported, got: %v", err)
			}
		})
	})
}

// MonotonicNonDecreasing carries state across invocations — it remembers the
// previous reading — so the first call can never fail and a refused read must
// not poison the watermark.
func TestMonotonicNonDecreasingBranches(t *testing.T) {
	t.Parallel()

	t.Run("the first reading establishes the baseline", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			l := &law.MonotonicNonDecreasing[int, int]{
				Read: func(*rapid.T, int) (int, error) { return 5, nil },
				Less: func(a, b int) bool { return a < b },
			}
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("the first reading cannot violate monotonicity: %v", err)
			}
		})
	})

	t.Run("a decreasing reading is flagged", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			vals := []int{9, 3}
			i := 0
			l := &law.MonotonicNonDecreasing[int, int]{
				Read: func(*rapid.T, int) (int, error) {
					v := vals[min(i, len(vals)-1)]
					i++
					return v, nil
				},
				Less: func(a, b int) bool { return a < b },
			}
			_ = l.Check(rt, 0, 0) // primes the watermark at 9
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("a reading below the previous one is a violation")
			}
		})
	})

	t.Run("a refused read holds vacuously", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			l := &law.MonotonicNonDecreasing[int, int]{
				Read: func(*rapid.T, int) (int, error) { return 0, errors.New("unavailable") },
				Less: func(a, b int) bool { return a < b },
			}
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a refused read is a precondition: %v", err)
			}
		})
	})
}

// The reader laws are self-consistency properties: they call the subject twice
// and compare. A read that fails both times is a precondition; one that
// changes its mind between calls is the violation.
func TestReaderLawSelfConsistency(t *testing.T) {
	t.Parallel()

	t.Run("Cacheable holds vacuously when both reads fail", func(t *testing.T) {
		t.Parallel()
		l := law.Cacheable[int, string, int]{
			Read: func(*rapid.T, int, string) (int, error) { return 0, errors.New("absent") },
			Keys: rapid.Just("k"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a consistently absent key is a precondition: %v", err)
			}
		})
	})

	t.Run("Cacheable flags a key that appears between reads", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			reads := 0
			l := law.Cacheable[int, string, int]{
				Read: func(*rapid.T, int, string) (int, error) {
					reads++
					if reads == 1 {
						return 0, errors.New("miss")
					}
					return 1, nil
				},
				Keys: rapid.Just("k"),
			}
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("a cached read that changes presence is a violation")
			}
		})
	})

	t.Run("Cacheable flags a value that changes between reads", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			reads := 0
			l := law.Cacheable[int, string, int]{
				Read: func(*rapid.T, int, string) (int, error) { reads++; return reads, nil },
				Keys: rapid.Just("k"),
			}
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("a cached read that returns a new value is a violation")
			}
		})
	})

	// A method that documents a default on error must actually return it —
	// returning a partially-populated value alongside the error is the defect.
	t.Run("DefaultOnError flags a non-default value beside an error", func(t *testing.T) {
		t.Parallel()
		l := law.DefaultOnError[int, string, int]{
			Read:    func(*rapid.T, int, string) (int, error) { return 42, errors.New("failed") },
			Default: 0,
			Eq:      func(a, b int) bool { return a == b },
			Keys:    rapid.Just("k"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("an error paired with a non-default value is a violation")
			}
		})
	})

	t.Run("DefaultOnError passes on success and on a proper default", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			ok := law.DefaultOnError[int, string, int]{
				Read:    func(*rapid.T, int, string) (int, error) { return 7, nil },
				Default: 0,
				Eq:      func(a, b int) bool { return a == b },
				Keys:    rapid.Just("k"),
			}
			if err := ok.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a successful read says nothing about the default: %v", err)
			}

			proper := ok
			proper.Read = func(*rapid.T, int, string) (int, error) { return 0, errors.New("failed") }
			if err := proper.Check(rt, 0, 0); err != nil {
				rt.Fatalf("an error paired with the default must pass: %v", err)
			}
		})
	})

	// Sticky remembers the first value it saw per key across invocations, so a
	// refused read must not poison that memory.
	t.Run("Sticky flags a value that changes after first resolution", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			reads := 0
			l := &law.Sticky[int, string, int]{
				Read: func(*rapid.T, int, string) (int, error) { reads++; return reads, nil },
				Eq:   func(a, b int) bool { return a == b },
				Keys: rapid.Just("k"),
			}
			_ = l.Check(rt, 0, 0) // records first
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("a sticky value that changes is a violation")
			}
		})
	})

	t.Run("Sticky holds vacuously when the read is refused", func(t *testing.T) {
		t.Parallel()
		l := &law.Sticky[int, string, int]{
			Read: func(*rapid.T, int, string) (int, error) { return 0, errors.New("absent") },
			Eq:   func(a, b int) bool { return a == b },
			Keys: rapid.Just("k"),
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a refused read is a precondition: %v", err)
			}
		})
	})
}
