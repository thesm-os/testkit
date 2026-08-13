// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/action"
)

// ErrGone stands in for a declaration's stamped miss sentinel.
var errGone = errors.New("opts_test: gone")

// missPair is a SUT/ref pair distinguished only by which error a miss
// answers: the identity hole the sentinel option exists to close.
type missPair struct{ err error }

func (p *missPair) read(_ context.Context, _ string) (string, error) {
	return "", p.err
}

// runReader drives one Reader step over the pair and returns its verdict.
func runReader(t *testing.T, sut, ref *missPair, options ...action.Opt) error {
	t.Helper()
	var got error
	rapid.Check(t, func(rt *rapid.T) {
		a := action.Reader("Read", rapid.Just("k"),
			func(ctx context.Context, p *missPair, k string) (string, error) {
				_ = k
				return p.read(ctx, k)
			}, options...)
		// The action compares the two subjects the runner would hand it.
		res := a.Run(rt, sut, ref)
		got = res.Err
	})
	return got
}

// TestWithSentinel pins the identity half of the reader comparison: armed,
// a private error beside the declared sentinel is a divergence; unarmed,
// presence is still the whole comparison, because a reference that mints
// its own errors must not fail a correct subject.
func TestWithSentinel(t *testing.T) {
	t.Parallel()

	t.Run("a wrong identity diverges when armed", func(t *testing.T) {
		t.Parallel()
		err := runReader(t,
			&missPair{err: errors.New("private miss")},
			&missPair{err: errGone},
			action.WithSentinel(errGone))
		if err == nil || !strings.Contains(err.Error(), "disagree on its identity") {
			t.Fatalf("a private error beside the declared sentinel must diverge, got: %v", err)
		}
	})

	t.Run("an agreeing identity passes when armed", func(t *testing.T) {
		t.Parallel()
		err := runReader(t,
			&missPair{err: fmt.Errorf("wrapped: %w", errGone)},
			&missPair{err: errGone},
			action.WithSentinel(errGone))
		if err != nil {
			t.Fatalf("a wrapped sentinel is the sentinel, got: %v", err)
		}
	})

	t.Run("unarmed stays presence-only", func(t *testing.T) {
		t.Parallel()
		err := runReader(t,
			&missPair{err: errors.New("private miss")},
			&missPair{err: errGone})
		if err != nil {
			t.Fatalf("without a sentinel the comparison is presence, got: %v", err)
		}
	})

	t.Run("the multi and batch readers carry the same guard", func(t *testing.T) {
		t.Parallel()
		var multiErr, batchErr error
		rapid.Check(t, func(rt *rapid.T) {
			sut := &missPair{err: errors.New("private miss")}
			ref := &missPair{err: errGone}
			m := action.MultiReader("Read", rapid.Just("k"),
				func(_ context.Context, p *missPair, _ string) (string, string, error) {
					return "", "", p.err
				}, action.WithSentinel(errGone))
			multiErr = m.Run(rt, sut, ref).Err
			b := action.BatchReader("ReadAll", rapid.Just("k"),
				func(_ context.Context, p *missPair, _ ...string) ([]string, error) {
					return nil, p.err
				}, action.WithSentinel(errGone))
			batchErr = b.Run(rt, sut, ref).Err
		})
		if multiErr == nil || batchErr == nil {
			t.Fatalf("every error-answering reader shape checks identity: multi=%v batch=%v",
				multiErr, batchErr)
		}
	})
}
