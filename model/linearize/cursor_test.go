// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize_test

import (
	"errors"
	"testing"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/linearize"
)

func TestCursor(t *testing.T) {
	t.Parallel()

	closedSentinel := errors.New("closed")

	t.Run("Next yields items in order until exhaustion", func(t *testing.T) {
		t.Parallel()
		m := linearize.Cursor[string]([]string{"a", "b"}, closedSentinel)
		history := []porcupine.Operation{
			opIO(0, "Next", nil, linearize.ReaderBoolResult[string]{Value: "a", OK: true}),
			opIO(1, "Next", nil, linearize.ReaderBoolResult[string]{Value: "b", OK: true}),
			opIO(2, "Next", nil, linearize.ReaderBoolResult[string]{OK: false}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "ordered drain")
	})

	t.Run("Next with wrong value is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.Cursor[string]([]string{"a"}, closedSentinel)
		history := []porcupine.Operation{
			opIO(0, "Next", nil, linearize.ReaderBoolResult[string]{Value: "wrong", OK: true}),
		}
		testkit.False(t, porcupine.CheckOperations(m, history), "wrong value rejected")
	})

	t.Run("Close idempotent then Next signals end", func(t *testing.T) {
		t.Parallel()
		m := linearize.Cursor[string]([]string{"a"}, closedSentinel)
		history := []porcupine.Operation{
			opIO(0, "Close", nil, linearize.WriterResult{}),
			opIO(1, "Close", nil, linearize.WriterResult{}),
			opIO(2, "Next", nil, linearize.ReaderBoolResult[string]{OK: false}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "double-close + post-close Next")
	})

	t.Run("Close returning an error is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.Cursor[string]([]string{"a"}, closedSentinel)
		history := []porcupine.Operation{
			opIO(0, "Close", nil, linearize.WriterResult{Err: errors.New("nope")}),
		}
		testkit.False(t, porcupine.CheckOperations(m, history), "Close must not error")
	})
}
