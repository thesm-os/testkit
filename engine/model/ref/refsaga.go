// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// provides the [CompensatingSaga] reference for the
// Saga composite-tier shape. Run executes steps in order until
// either all succeed (committed) or one fails (compensated): the
// reference invokes the compensator for each successfully-completed
// prior step in reverse order.

package ref

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// SagaStep is one step of the saga. Run is the forward action;
// Compensate is the reverse action invoked when a later step
// fails.
type SagaStep[T any] struct {
	// Name identifies the step in diagnostics.
	Name string

	// Run executes the forward action. A nil error advances the
	// saga; any error triggers compensation of prior steps.
	Run func(context.Context, T) error

	// Compensate reverses this step's effects. Invoked in reverse
	// order over the steps that completed successfully before the
	// failing step.
	Compensate func(context.Context, T) error
}

// CompensatingSaga executes a fixed sequence of steps, compensating
// on first failure. Construct with [NewCompensatingSaga].
type CompensatingSaga[T any] struct {
	mu    sync.Mutex
	steps []SagaStep[T]
}

// NewCompensatingSaga constructs a saga over the given steps.
func NewCompensatingSaga[T any](steps []SagaStep[T]) *CompensatingSaga[T] {
	return &CompensatingSaga[T]{steps: steps}
}

// Run executes the saga against the supplied subject. Returns:
//
//   - nil on full success (every Run returned nil).
//   - The wrapped step error when a Run fails and compensation
//     succeeds.
//   - A joined error when both a Run and one of its compensations
//     fail; both errors are preserved via errors.Join.
func (s *CompensatingSaga[T]) Run(ctx context.Context, subject T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, step := range s.steps {
		if err := step.Run(ctx, subject); err != nil {
			compErr := s.compensate(ctx, subject, i-1)
			if compErr != nil {
				return errors.Join(
					fmt.Errorf("saga step %d (%s) failed: %w", i, step.Name, err),
					compErr,
				)
			}
			return fmt.Errorf("saga step %d (%s) failed: %w", i, step.Name, err)
		}
	}
	return nil
}

// compensate invokes the Compensate of each step in reverse order
// from lastCompleted down to 0. Returns a joined error of every
// compensation failure (nil when all compensations succeed).
func (s *CompensatingSaga[T]) compensate(ctx context.Context, subject T, lastCompleted int) error {
	var errs []error
	for j := lastCompleted; j >= 0; j-- {
		if s.steps[j].Compensate == nil {
			continue
		}
		if err := s.steps[j].Compensate(ctx, subject); err != nil {
			errs = append(errs, fmt.Errorf("compensate step %d (%s): %w", j, s.steps[j].Name, err))
		}
	}
	return errors.Join(errs...)
}
