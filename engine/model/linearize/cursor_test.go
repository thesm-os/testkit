// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize_test

import (
	"errors"
	"testing"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/linearize"
)

var errCursorClosed = errors.New("cursor closed")

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

// Driving the model's fields directly reaches the arms a well-formed history
// never produces: mis-typed results, unknown operations, and the state
// helpers porcupine only calls while deduplicating or rendering a failure.
func TestCursorModelDefensiveArms(t *testing.T) {
	t.Parallel()

	items := []string{"a", "b"}
	newModel := func() porcupine.Model { return linearize.Cursor(items, errCursorClosed) }
	in := func(name string) model.OpInput { return model.OpInput{Name: name} }
	out := func(r any) model.OpOutput { return model.OpOutput{Result: r} }

	t.Run("Next with a non-reader result is rejected", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		ok, _ := m.Step(m.Init(), in("Next"), out("nonsense"))
		testkit.False(t, ok, "a mis-typed Next output cannot be linearizable")
	})

	t.Run("Close with a non-writer result is rejected", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		ok, _ := m.Step(m.Init(), in("Close"), out("nonsense"))
		testkit.False(t, ok, "a mis-typed Close output cannot be linearizable")
	})

	t.Run("an unknown operation is rejected", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		ok, _ := m.Step(m.Init(), in("Rewind"), out(nil))
		testkit.False(t, ok, "an unmodelled operation cannot be accepted")
	})

	t.Run("a failing Close is rejected", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		ok, _ := m.Step(m.Init(), in("Close"), out(linearize.WriterResult{Err: errCursorClosed}))
		testkit.False(t, ok, "Close is not permitted to fail in this model")
	})

	t.Run("Next after Close must report exhaustion", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		_, closed := m.Step(m.Init(), in("Close"), out(linearize.WriterResult{}))

		ok, _ := m.Step(closed, in("Next"), out(linearize.ReaderBoolResult[string]{OK: false}))
		testkit.True(t, ok, "a closed cursor yields OK=false")

		ok, _ = m.Step(closed, in("Next"), out(linearize.ReaderBoolResult[string]{OK: true, Value: "a"}))
		testkit.False(t, ok, "a closed cursor must not keep producing values")
	})

	t.Run("Next past the end must report exhaustion", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		s := m.Init()
		for _, want := range items {
			var ok bool
			ok, s = m.Step(s, in("Next"), out(linearize.ReaderBoolResult[string]{OK: true, Value: want}))
			testkit.True(t, ok, "each item is produced in order")
		}
		ok, _ := m.Step(s, in("Next"), out(linearize.ReaderBoolResult[string]{OK: false}))
		testkit.True(t, ok, "an exhausted cursor yields OK=false")
	})

	t.Run("Next yielding the wrong value is rejected", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		ok, _ := m.Step(m.Init(), in("Next"), out(linearize.ReaderBoolResult[string]{OK: true, Value: "z"}))
		testkit.False(t, ok, "the cursor must produce items in order")
	})

	t.Run("Next reporting exhaustion too early is rejected", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		ok, _ := m.Step(m.Init(), in("Next"), out(linearize.ReaderBoolResult[string]{OK: false}))
		testkit.False(t, ok, "an open cursor with items left must produce one")
	})

	t.Run("state helpers render position and status", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		open := m.Init()
		_, advanced := m.Step(open, in("Next"), out(linearize.ReaderBoolResult[string]{OK: true, Value: "a"}))
		_, closed := m.Step(open, in("Close"), out(linearize.WriterResult{}))

		testkit.True(t, m.Equal(open, m.Init()), "two fresh cursors are equal")
		testkit.False(t, m.Equal(open, advanced), "position is part of identity")
		testkit.False(t, m.Equal(open, closed), "closed-ness is part of identity")

		testkit.Contains(t, m.DescribeState(open), "open", "open cursor")
		testkit.Contains(t, m.DescribeState(closed), "closed", "closed cursor")
		testkit.Contains(t, m.DescribeOperation(in("Next"), out(nil)), "Next", "names the operation")
	})
}
