// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
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
