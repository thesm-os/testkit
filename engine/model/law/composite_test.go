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

type poolSUT struct {
	gets, puts, outstanding int
	balanced                bool
}

func TestPoolBalancedGetPutLaw(t *testing.T) {
	t.Parallel()

	t.Run("consistent stats pass", func(t *testing.T) {
		t.Parallel()
		s := &poolSUT{gets: 5, puts: 3, outstanding: 2}
		l := law.PoolBalancedGetPut[*poolSUT]{
			Stats: func(_ *rapid.T, p *poolSUT) (int, int, int) { return p.gets, p.puts, p.outstanding },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("negative outstanding flagged", func(t *testing.T) {
		t.Parallel()
		s := &poolSUT{gets: 3, puts: 5, outstanding: -2}
		l := law.PoolBalancedGetPut[*poolSUT]{
			Stats: func(_ *rapid.T, p *poolSUT) (int, int, int) { return p.gets, p.puts, p.outstanding },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected negative-outstanding flagged")
			}
		})
	})

	t.Run("inconsistent gets-puts-outstanding flagged", func(t *testing.T) {
		t.Parallel()
		s := &poolSUT{gets: 5, puts: 0, outstanding: 99}
		l := law.PoolBalancedGetPut[*poolSUT]{
			Stats: func(_ *rapid.T, p *poolSUT) (int, int, int) { return p.gets, p.puts, p.outstanding },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected inconsistency flagged")
			}
		})
	})
}

func TestPoolLeakFree(t *testing.T) {
	t.Parallel()

	t.Run("balanced=true passes", func(t *testing.T) {
		t.Parallel()
		s := &poolSUT{balanced: true}
		l := law.PoolLeakFree[*poolSUT]{
			Balanced: func(_ *rapid.T, p *poolSUT) bool { return p.balanced },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("balanced=false flagged", func(t *testing.T) {
		t.Parallel()
		s := &poolSUT{balanced: false}
		l := law.PoolLeakFree[*poolSUT]{
			Balanced: func(_ *rapid.T, p *poolSUT) bool { return p.balanced },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected leak flagged")
			}
		})
	})
}

func TestCursorCloseIdempotentLaw(t *testing.T) {
	t.Parallel()

	// The marker is the runner's dispatch, on the value receiver: a registry
	// holding the law by value must still route it to a throwaway pair.
	var iso law.Isolated = law.CursorCloseIdempotent[*lifecycleSUT]{}
	iso.IsolatedLaw()

	t.Run("two no-op closes pass", func(t *testing.T) {
		t.Parallel()
		l := law.CursorCloseIdempotent[*lifecycleSUT]{
			Close: func(_ *rapid.T, _ *lifecycleSUT) error { return nil },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("second close that errors is flagged", func(t *testing.T) {
		t.Parallel()
		call := 0
		l := law.CursorCloseIdempotent[*lifecycleSUT]{
			Close: func(_ *rapid.T, _ *lifecycleSUT) error {
				call++
				if call > 1 {
					return errors.New("nope")
				}
				return nil
			},
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{}); err == nil {
				rt.Fatal("expected second-close error flagged")
			}
		})
	})
}

func TestCursorNextAfterCloseSentinel(t *testing.T) {
	t.Parallel()

	// The marker is the runner's dispatch, on the value receiver: a registry
	// holding the law by value must still route it to a throwaway pair.
	var iso law.Isolated = law.CursorNextAfterCloseSentinel[*lifecycleSUT, int]{}
	iso.IsolatedLaw()

	sentinel := errors.New("closed")

	t.Run("Next-after-Close returns sentinel passes", func(t *testing.T) {
		t.Parallel()
		l := law.CursorNextAfterCloseSentinel[*lifecycleSUT, string]{
			Close:    func(_ *rapid.T, _ *lifecycleSUT) error { return nil },
			Next:     func(_ *rapid.T, _ *lifecycleSUT) (string, bool, error) { return "", false, sentinel },
			Sentinel: sentinel,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("Next-after-Close returning nil flagged", func(t *testing.T) {
		t.Parallel()
		l := law.CursorNextAfterCloseSentinel[*lifecycleSUT, string]{
			Close:    func(_ *rapid.T, _ *lifecycleSUT) error { return nil },
			Next:     func(_ *rapid.T, _ *lifecycleSUT) (string, bool, error) { return "", false, nil },
			Sentinel: sentinel,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, &lifecycleSUT{}, &lifecycleSUT{}); err == nil {
				rt.Fatal("expected sentinel mismatch")
			}
		})
	})
}

type txState struct {
	committed bool
}

func TestTwoPhaseNoRollbackAfterCommit(t *testing.T) {
	t.Parallel()

	closed := errors.New("tx closed")

	t.Run("rollback-after-commit returns closed sentinel passes", func(t *testing.T) {
		t.Parallel()
		l := law.TwoPhaseNoRollbackAfterCommit[any, *txState]{
			Begin:  func(_ *rapid.T, _ any) (*txState, error) { return &txState{}, nil },
			Commit: func(_ *rapid.T, _ any, tx *txState) error { tx.committed = true; return nil },
			Rollback: func(_ *rapid.T, _ any, tx *txState) error {
				if tx.committed {
					return closed
				}
				return nil
			},
			Closed: closed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("rollback-after-commit returning nil flagged", func(t *testing.T) {
		t.Parallel()
		l := law.TwoPhaseNoRollbackAfterCommit[any, *txState]{
			Begin:    func(_ *rapid.T, _ any) (*txState, error) { return &txState{}, nil },
			Commit:   func(_ *rapid.T, _ any, _ *txState) error { return nil },
			Rollback: func(_ *rapid.T, _ any, _ *txState) error { return nil }, // permissive — bug
			Closed:   closed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, nil, nil); err == nil {
				rt.Fatal("expected permissive rollback flagged")
			}
		})
	})
}

type sagaState struct {
	count int
}

func TestSagaFullCompensation(t *testing.T) {
	t.Parallel()

	t.Run("error with full compensation passes", func(t *testing.T) {
		t.Parallel()
		s := &sagaState{count: 0}
		l := law.SagaFullCompensation[*sagaState, int]{
			Run:     func(_ *rapid.T, _ *sagaState) error { return errors.New("step failed") },
			Observe: func(_ *rapid.T, st *sagaState) int { return st.count },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("error leaving residue flagged", func(t *testing.T) {
		t.Parallel()
		s := &sagaState{count: 0}
		l := law.SagaFullCompensation[*sagaState, int]{
			Run: func(_ *rapid.T, st *sagaState) error {
				st.count++ // mutates state, then errors → no compensation
				return errors.New("step failed")
			},
			Observe: func(_ *rapid.T, st *sagaState) int { return st.count },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err == nil {
				rt.Fatal("expected residue flagged")
			}
		})
	})

	t.Run("successful Run is vacuous", func(t *testing.T) {
		t.Parallel()
		s := &sagaState{}
		l := law.SagaFullCompensation[*sagaState, int]{
			Run:     func(_ *rapid.T, _ *sagaState) error { return nil },
			Observe: func(_ *rapid.T, st *sagaState) int { return st.count },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, s, s); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

// The composite laws all assert that a resource, once finalised, rejects
// further use with a specific sentinel. A subject that refuses the *setup*
// has failed a precondition; one that accepts it and then allows the
// forbidden follow-up has violated the law.
func TestCompositeLawPreconditionsAndSentinels(t *testing.T) {
	t.Parallel()

	closed := errors.New("closed")

	t.Run("CursorNextAfterCloseSentinel holds vacuously when Close is refused", func(t *testing.T) {
		t.Parallel()
		l := law.CursorNextAfterCloseSentinel[int, string]{
			Close:    func(*rapid.T, int) error { return errors.New("busy") },
			Next:     func(*rapid.T, int) (string, bool, error) { return "", false, closed },
			Sentinel: closed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); !law.Holds(err) {
				rt.Fatalf("a cursor that cannot be closed is a precondition: %v", err)
			}
		})
	})

	t.Run("CursorNextAfterCloseSentinel flags a cursor that keeps yielding", func(t *testing.T) {
		t.Parallel()
		l := law.CursorNextAfterCloseSentinel[int, string]{
			Close:    func(*rapid.T, int) error { return nil },
			Next:     func(*rapid.T, int) (string, bool, error) { return "still here", true, nil },
			Sentinel: closed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("a closed cursor must not keep producing values")
			}
		})
	})

	t.Run("TwoPhaseNoRollbackAfterCommit separates its preconditions", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(rt *rapid.T) {
			noBegin := law.TwoPhaseNoRollbackAfterCommit[int, int]{
				Begin:    func(*rapid.T, int) (int, error) { return 0, errors.New("no tx") },
				Commit:   func(*rapid.T, int, int) error { return nil },
				Rollback: func(*rapid.T, int, int) error { return closed },
				Closed:   closed,
			}
			if err := noBegin.Check(rt, 0, 0); !law.Holds(err) {
				rt.Fatalf("a refused Begin is a precondition: %v", err)
			}

			noCommit := noBegin
			noCommit.Begin = func(*rapid.T, int) (int, error) { return 0, nil }
			noCommit.Commit = func(*rapid.T, int, int) error { return errors.New("conflict") }
			if err := noCommit.Check(rt, 0, 0); !law.Holds(err) {
				rt.Fatalf("a refused Commit is a precondition: %v", err)
			}
		})
	})

	t.Run("TwoPhaseNoRollbackAfterCommit flags an accepted rollback", func(t *testing.T) {
		t.Parallel()
		l := law.TwoPhaseNoRollbackAfterCommit[int, int]{
			Begin:    func(*rapid.T, int) (int, error) { return 0, nil },
			Commit:   func(*rapid.T, int, int) error { return nil },
			Rollback: func(*rapid.T, int, int) error { return nil }, // wrongly succeeds
			Closed:   closed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err == nil {
				rt.Fatal("rolling back a committed transaction must be rejected")
			}
		})
	})

	// Which terminal operation runs first is drawn, so the law must reject the
	// *other* one whichever way the draw went — and the diagnostic has to name
	// the pair correctly.
	t.Run("TwoPhaseCommitOrRollback rejects the second terminal operation", func(t *testing.T) {
		t.Parallel()
		l := law.TwoPhaseCommitOrRollback[int, int]{
			Begin:    func(*rapid.T, int) (int, error) { return 0, nil },
			Commit:   func(*rapid.T, int, int) error { return nil },
			Rollback: func(*rapid.T, int, int) error { return nil }, // neither closes the tx
			Closed:   closed,
		}
		seen := map[string]bool{}
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, 0, 0)
			if err == nil {
				rt.Fatal("a transaction must accept only one terminal operation")
			}
			switch {
			case strings.Contains(err.Error(), "commit after rollback"):
				seen["commit-after-rollback"] = true
			case strings.Contains(err.Error(), "rollback after commit"):
				seen["rollback-after-commit"] = true
			}
		})
		if len(seen) < 2 {
			t.Fatalf("both draw orders must be exercised, saw %v", seen)
		}
	})

	t.Run("TwoPhaseCommitOrRollback holds vacuously when the first op fails", func(t *testing.T) {
		t.Parallel()
		l := law.TwoPhaseCommitOrRollback[int, int]{
			Begin:    func(*rapid.T, int) (int, error) { return 0, nil },
			Commit:   func(*rapid.T, int, int) error { return errors.New("conflict") },
			Rollback: func(*rapid.T, int, int) error { return errors.New("conflict") },
			Closed:   closed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); err != nil {
				rt.Fatalf("a failed first terminal op is a precondition: %v", err)
			}
		})
	})

	t.Run("TwoPhaseCommitOrRollback holds vacuously when Begin is refused", func(t *testing.T) {
		t.Parallel()
		l := law.TwoPhaseCommitOrRollback[int, int]{
			Begin:    func(*rapid.T, int) (int, error) { return 0, errors.New("no transactions") },
			Commit:   func(*rapid.T, int, int) error { return nil },
			Rollback: func(*rapid.T, int, int) error { return nil },
			Closed:   closed,
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, 0, 0); !law.Holds(err) {
				rt.Fatalf("a subject that cannot begin a transaction is a precondition: %v", err)
			}
		})
	})
}

// TestSagaFullCompensationPair holds the mirrored saga to the conduct
// contract: a completed run lands on both sides, and a reference that
// refuses it is reported.
func TestSagaFullCompensationPair(t *testing.T) {
	t.Parallel()

	t.Run("the completed run lands on both sides", func(t *testing.T) {
		t.Parallel()
		l := law.SagaFullCompensation[*sagaState, int]{
			Run:     func(_ *rapid.T, s *sagaState) error { s.count++; return nil },
			Observe: func(_ *rapid.T, s *sagaState) int { return s.count },
		}
		rapid.Check(t, func(rt *rapid.T) {
			sut, refS := &sagaState{}, &sagaState{}
			if err := l.Check(rt, sut, refS); err != nil {
				rt.Fatal(err)
			}
			if refS.count == 0 {
				rt.Fatal("the reference never saw the run: the pair has diverged")
			}
		})
	})

	t.Run("a refusing reference is tolerated", func(t *testing.T) {
		t.Parallel()
		// Run draws its own work, so the two sides' draws differ by design:
		// the reference refusing its own run is a compensated no-op, not a
		// disagreement — reporting it failed correct pairs on the first
		// colliding draw.
		rapid.Check(t, func(rt *rapid.T) {
			refS := &sagaState{}
			l := law.SagaFullCompensation[*sagaState, int]{
				Run: func(_ *rapid.T, s *sagaState) error {
					if s == refS {
						return errors.New("refused")
					}
					s.count++
					return nil
				},
				Observe: func(_ *rapid.T, s *sagaState) int { return s.count },
			}
			if err := l.Check(rt, &sagaState{}, refS); err != nil {
				rt.Fatalf("a reference refusing its own drawn run is not a disagreement: %v", err)
			}
		})
	})
}
