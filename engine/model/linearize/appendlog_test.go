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

func TestAppendLog(t *testing.T) {
	t.Parallel()

	t.Run("Append-then-At-then-Len is linearizable", func(t *testing.T) {
		t.Parallel()
		m := linearize.AppendLog[string]()
		history := []porcupine.Operation{
			opIO(0, "Append", "a", linearize.AppendResult{Off: 0}),
			opIO(1, "Append", "b", linearize.AppendResult{Off: 1}),
			opIO(2, "At", int64(1), linearize.ReaderResult[string]{Value: "b"}),
			opIO(3, "Len", nil, int64(2)),
		}
		testkit.True(t, porcupine.CheckOperations(m, history),
			"matching append-log history should be linearizable")
	})

	t.Run("a refused append leaves the log", func(t *testing.T) {
		t.Parallel()
		m := linearize.AppendLog[string]()
		history := []porcupine.Operation{
			opIO(0, "Append", "a", linearize.AppendResult{Err: errors.New("full")}),
			opIO(1, "Append", "b", linearize.AppendResult{Off: 0}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history),
			"the refusal appended nothing, so the next offset is still zero")
	})

	t.Run("non-monotonic Append offset is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.AppendLog[string]()
		history := []porcupine.Operation{
			opIO(0, "Append", "a", linearize.AppendResult{Off: 0}),
			opIO(1, "Append", "b", linearize.AppendResult{Off: 99}),
		}
		testkit.False(t, porcupine.CheckOperations(m, history),
			"non-contiguous offset should be rejected")
	})

	t.Run("At out-of-range yields error", func(t *testing.T) {
		t.Parallel()
		m := linearize.AppendLog[string]()
		history := []porcupine.Operation{
			opIO(0, "At", int64(0), linearize.ReaderResult[string]{Err: errors.New("oob")}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history),
			"out-of-range At returning an error is valid on an empty log")
	})

	t.Run("At with wrong value is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.AppendLog[string]()
		history := []porcupine.Operation{
			opIO(0, "Append", "a", linearize.AppendResult{Off: 0}),
			opIO(1, "At", int64(0), linearize.ReaderResult[string]{Value: "wrong"}),
		}
		testkit.False(t, porcupine.CheckOperations(m, history),
			"wrong-value At should be rejected")
	})

	t.Run("Len with wrong count is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.AppendLog[string]()
		history := []porcupine.Operation{
			opIO(0, "Append", "a", linearize.AppendResult{Off: 0}),
			opIO(1, "Len", nil, int64(99)),
		}
		testkit.False(t, porcupine.CheckOperations(m, history),
			"wrong Len should be rejected")
	})
}

// A log's offsets are its contract: Append must return the index it wrote at,
// so an implementation that returns a running total or a timestamp is a real
// defect. These drive that plus the mis-typed and out-of-range arms.
func TestAppendLogModelBranches(t *testing.T) {
	t.Parallel()

	m := linearize.AppendLog[string]()
	in := func(name string, args any) model.OpInput { return model.OpInput{Name: name, Args: args} }
	out := func(r any) model.OpOutput { return model.OpOutput{Result: r} }

	t.Run("Append must return the index it wrote at", func(t *testing.T) {
		t.Parallel()
		ok, next := m.Step(m.Init(), in("Append", "a"), out(linearize.AppendResult{Off: 0}))
		testkit.True(t, ok, "the first append lands at offset 0")
		ok, _ = m.Step(next, in("Append", "b"), out(linearize.AppendResult{Off: 1}))
		testkit.True(t, ok, "the second append lands at offset 1")

		ok, _ = m.Step(next, in("Append", "b"), out(linearize.AppendResult{Off: 7}))
		testkit.False(t, ok, "an offset that is not the current length is wrong")
	})

	t.Run("Append rejects mis-typed args and results", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Append", 42), out(linearize.AppendResult{Off: 0}))
		testkit.False(t, ok, "the value must match the log's element type")
		ok, _ = m.Step(m.Init(), in("Append", "a"), out("zero"))
		testkit.False(t, ok, "the offset must be an int64")
	})

	t.Run("At returns the element at an in-range index", func(t *testing.T) {
		t.Parallel()
		_, one := m.Step(m.Init(), in("Append", "a"), out(linearize.AppendResult{Off: 0}))
		ok, _ := m.Step(one, in("At", int64(0)), out(linearize.ReaderResult[string]{Value: "a"}))
		testkit.True(t, ok, "index 0 holds the first appended value")
		ok, _ = m.Step(one, in("At", int64(0)), out(linearize.ReaderResult[string]{Value: "z"}))
		testkit.False(t, ok, "a different value at that index is wrong")
	})

	// Out-of-range reads must error rather than return a zero value, at both
	// ends — a negative index is as invalid as one past the end.
	t.Run("At out of range must error", func(t *testing.T) {
		t.Parallel()
		_, one := m.Step(m.Init(), in("Append", "a"), out(linearize.AppendResult{Off: 0}))
		for _, idx := range []int64{-1, 5} {
			ok, _ := m.Step(one, in("At", idx), out(linearize.ReaderResult[string]{Err: errors.New("range")}))
			testkit.True(t, ok, "an out-of-range read errors")
			ok, _ = m.Step(one, in("At", idx), out(linearize.ReaderResult[string]{Value: "a"}))
			testkit.False(t, ok, "an out-of-range read must not succeed")
		}
	})

	t.Run("At rejects mis-typed args and results", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("At", "zero"), out(linearize.ReaderResult[string]{}))
		testkit.False(t, ok, "the index must be an int64")
		ok, _ = m.Step(m.Init(), in("At", int64(0)), out("nonsense"))
		testkit.False(t, ok, "the result must be a ReaderResult")
	})

	t.Run("Len reports the element count", func(t *testing.T) {
		t.Parallel()
		_, one := m.Step(m.Init(), in("Append", "a"), out(linearize.AppendResult{Off: 0}))
		ok, _ := m.Step(one, in("Len", nil), out(int64(1)))
		testkit.True(t, ok, "one element means length 1")
		ok, _ = m.Step(one, in("Len", nil), out(int64(9)))
		testkit.False(t, ok, "a wrong length is rejected")
		ok, _ = m.Step(one, in("Len", nil), out("one"))
		testkit.False(t, ok, "the length must be an int64")
	})

	t.Run("an unknown operation is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Truncate", nil), out(int64(0)))
		testkit.False(t, ok, "an unmodelled operation cannot be accepted")
	})

	t.Run("state helpers", func(t *testing.T) {
		t.Parallel()
		empty := m.Init()
		_, one := m.Step(empty, in("Append", "a"), out(linearize.AppendResult{Off: 0}))
		testkit.True(t, m.Equal(empty, m.Init()), "two empty logs are equal")
		testkit.False(t, m.Equal(empty, one), "contents are the state")
		testkit.Equal(t, m.DescribeState(one), "len=1", "renders the length")
		testkit.Contains(t,
			m.DescribeOperation(in("Append", "a"), out(linearize.AppendResult{Off: 0})),
			"Append", "names the operation")
	})
}
