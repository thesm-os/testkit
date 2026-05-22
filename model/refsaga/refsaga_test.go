// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package refsaga_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/refsaga"
)

type subject struct {
	mu    chan struct{}
	trace []string
}

func newSubject() *subject {
	return &subject{mu: make(chan struct{}, 1)}
}

func (s *subject) record(name string) {
	s.mu <- struct{}{}
	s.trace = append(s.trace, name)
	<-s.mu
}

var errStepBoom = errors.New("step boom")

func TestCompensatingSaga(t *testing.T) {
	t.Parallel()

	stepRecord := func(name string) refsaga.Step[*subject] {
		return refsaga.Step[*subject]{
			Name:       name,
			Run:        func(_ context.Context, s *subject) error { s.record("run-" + name); return nil },
			Compensate: func(_ context.Context, s *subject) error { s.record("comp-" + name); return nil },
		}
	}

	t.Run("all steps succeed → trace shows forward order, no compensations", func(t *testing.T) {
		t.Parallel()
		s := newSubject()
		saga := refsaga.NewCompensatingSaga([]refsaga.Step[*subject]{
			stepRecord("A"),
			stepRecord("B"),
			stepRecord("C"),
		})
		testkit.NoError(t, saga.Run(t.Context(), s), "run")
		testkit.Equal(t, s.trace, []string{"run-A", "run-B", "run-C"}, "forward only")
	})

	t.Run("middle-step failure compensates prior steps in reverse", func(t *testing.T) {
		t.Parallel()
		s := newSubject()
		failingB := refsaga.Step[*subject]{
			Name: "B",
			Run:  func(_ context.Context, _ *subject) error { return errStepBoom },
		}
		saga := refsaga.NewCompensatingSaga([]refsaga.Step[*subject]{
			stepRecord("A"),
			failingB,
			stepRecord("C"),
		})
		err := saga.Run(t.Context(), s)
		testkit.True(t, errors.Is(err, errStepBoom), "step error surfaced")
		testkit.Equal(t, s.trace, []string{"run-A", "comp-A"}, "A run then compensated")
	})

	t.Run("first-step failure runs no compensations", func(t *testing.T) {
		t.Parallel()
		s := newSubject()
		failingA := refsaga.Step[*subject]{
			Name: "A",
			Run:  func(_ context.Context, _ *subject) error { return errStepBoom },
		}
		saga := refsaga.NewCompensatingSaga([]refsaga.Step[*subject]{
			failingA,
			stepRecord("B"),
		})
		err := saga.Run(t.Context(), s)
		testkit.True(t, errors.Is(err, errStepBoom), "step error")
		testkit.Equal(t, len(s.trace), 0, "no records")
	})

	t.Run("compensation failure joins with step failure", func(t *testing.T) {
		t.Parallel()
		errCompBoom := errors.New("comp boom")
		failingA := refsaga.Step[*subject]{
			Name:       "A",
			Run:        func(_ context.Context, _ *subject) error { return nil },
			Compensate: func(_ context.Context, _ *subject) error { return errCompBoom },
		}
		failingB := refsaga.Step[*subject]{
			Name: "B",
			Run:  func(_ context.Context, _ *subject) error { return errStepBoom },
		}
		saga := refsaga.NewCompensatingSaga([]refsaga.Step[*subject]{failingA, failingB})
		err := saga.Run(t.Context(), newSubject())
		testkit.True(t, errors.Is(err, errStepBoom), "step error joined")
		testkit.True(t, errors.Is(err, errCompBoom), "compensation error joined")
	})
}
