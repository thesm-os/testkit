// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Action errors are diagnostic (SUT vs ref comparison), not wrapped.
package action

import (
	"context"
	"fmt"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model"
)

// Pool creates an action for the Pool composite-tier shape: a
// Get/Put pair where every Get is followed by a matching Put. The
// action runs one full Get-then-Put cycle per invocation, comparing
// error outcomes on each step.
//
// The leak-free and balanced-counter contracts are observable
// across iterations as a law (the iteration count of Get must
// match Put's at quiescence).
func Pool[T, R any](
	name string,
	get func(context.Context, T) (R, error),
	put func(context.Context, T, R) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutR, sutErr := get(rt.Context(), sut)
			refR, refErr := get(rt.Context(), ref)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err: fmt.Errorf("%s.Get: SUT err=%v, ref err=%v", name, sutErr, refErr),
				}
			}
			if sutErr != nil {
				return model.ActionResult{}
			}
			sutPutErr := put(rt.Context(), sut, sutR)
			refPutErr := put(rt.Context(), ref, refR)
			if (sutPutErr == nil) != (refPutErr == nil) {
				return model.ActionResult{
					Err: fmt.Errorf("%s.Put: SUT err=%v, ref err=%v", name, sutPutErr, refPutErr),
				}
			}
			return model.ActionResult{}
		},
	}
}

// Cursor creates an action for the Cursor composite-tier shape: a
// Next/Close pair. The action drives Next until exhaustion or
// limit, then closes the cursor on both sides. NextLimit caps the
// drain to prevent runaway iteration on a buggy cursor.
//
// The close-idempotence and next-after-close-sentinel contracts
// are enforced by laws layered on top.
func Cursor[T, V any](
	name string,
	next func(context.Context, T) (V, bool, error),
	closeFn func(context.Context, T) error,
	nextLimit int,
) model.Action[T] {
	if nextLimit <= 0 {
		nextLimit = 100
	}
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			drain := func(impl T) (int, error) {
				count := 0
				for i := 0; i < nextLimit; i++ {
					_, ok, err := next(rt.Context(), impl)
					if err != nil {
						return count, err
					}
					if !ok {
						return count, nil
					}
					count++
				}
				return count, fmt.Errorf("%s: next limit %d exceeded", name, nextLimit)
			}

			sutCount, sutErr := drain(sut)
			refCount, refErr := drain(ref)
			// Drain errors (next-limit overflow, mid-drain failure)
			// are always faults: a cursor that doesn't terminate is
			// a bug regardless of impl agreement.
			if sutErr != nil || refErr != nil {
				return model.ActionResult{
					Err:    fmt.Errorf("%s.Next: SUT err=%v, ref err=%v", name, sutErr, refErr),
					Output: sutCount,
				}
			}
			if sutCount != refCount {
				return model.ActionResult{
					Err:    fmt.Errorf("%s.Next: SUT yielded %d, ref yielded %d", name, sutCount, refCount),
					Output: sutCount,
				}
			}
			sutCloseErr := closeFn(rt.Context(), sut)
			refCloseErr := closeFn(rt.Context(), ref)
			if (sutCloseErr == nil) != (refCloseErr == nil) {
				return model.ActionResult{
					Err: fmt.Errorf("%s.Close: SUT err=%v, ref err=%v", name, sutCloseErr, refCloseErr),
				}
			}
			return model.ActionResult{Output: sutCount}
		},
	}
}

// TwoPhase creates an action for the TwoPhase composite-tier shape:
// a Begin/Commit/Rollback triad where Begin returns a transaction
// handle, then either Commit or Rollback finalizes it. The action
// runs a full Begin → (Commit | Rollback) sequence per invocation
// and compares error outcomes at each step. The choice of
// commit-or-rollback is drawn from the rapid generator.
//
// The mutex-of-Commit-or-Rollback contract is enforced by laws
// over the trace.
func TwoPhase[T, Tx any](
	name string,
	begin func(context.Context, T) (Tx, error),
	commit func(context.Context, T, Tx) error,
	rollback func(context.Context, T, Tx) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutTx, sutErr := begin(rt.Context(), sut)
			refTx, refErr := begin(rt.Context(), ref)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err: fmt.Errorf("%s.Begin: SUT err=%v, ref err=%v", name, sutErr, refErr),
				}
			}
			if sutErr != nil {
				return model.ActionResult{}
			}
			doCommit := rapid.Bool().Draw(rt, name+"_commit")
			var sutFinErr, refFinErr error
			if doCommit {
				sutFinErr = commit(rt.Context(), sut, sutTx)
				refFinErr = commit(rt.Context(), ref, refTx)
			} else {
				sutFinErr = rollback(rt.Context(), sut, sutTx)
				refFinErr = rollback(rt.Context(), ref, refTx)
			}
			step := "Commit"
			if !doCommit {
				step = "Rollback"
			}
			if (sutFinErr == nil) != (refFinErr == nil) {
				return model.ActionResult{
					Err: fmt.Errorf("%s.%s: SUT err=%v, ref err=%v", name, step, sutFinErr, refFinErr),
				}
			}
			return model.ActionResult{Output: step}
		},
	}
}

// Saga creates an action for the Saga composite-tier shape: a
// chained sequence of N steps where each step's success carries
// state forward and any failure triggers reverse compensation
// over the prior steps.
//
// Steps is the ordered list of step functions; Compensate mirrors
// the steps list with one compensator per step. The action runs
// the full chain on both SUT and ref, compares the outcome
// (committed-through-step-N vs compensated-from-step-N), and
// records the divergent step on mismatch.
//
// Both side step lengths must equal len(Steps). On any mismatch
// the action returns an error naming the divergent step.
func Saga[T any](
	name string,
	steps []func(context.Context, T) error,
	compensate []func(context.Context, T) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			run := func(impl T) (failedAt int, compErr error) {
				for i, step := range steps {
					if err := step(rt.Context(), impl); err != nil {
						for j := i - 1; j >= 0; j-- {
							if cErr := compensate[j](rt.Context(), impl); cErr != nil {
								return i, fmt.Errorf(
									"step %d failed: %w; compensation %d also failed: %w",
									i, err, j, cErr,
								)
							}
						}
						return i, err
					}
				}
				return -1, nil
			}

			sutFailedAt, sutErr := run(sut)
			refFailedAt, refErr := run(ref)
			if sutFailedAt != refFailedAt {
				return model.ActionResult{
					Err: fmt.Errorf("%s: SUT failed at step %d, ref failed at step %d (sutErr=%v, refErr=%v)",
						name, sutFailedAt, refFailedAt, sutErr, refErr),
					Output: sutFailedAt,
				}
			}
			// failedAt already encodes error nullity — run returns -1 with a
			// nil error or a step index with a non-nil one — so agreeing on
			// the step is agreeing on whether the saga failed.
			return model.ActionResult{Output: sutFailedAt}
		},
	}
}
