// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize_test

import (
	"testing"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/linearize"
)

// opIO constructs a porcupine.Operation pair (input + output) for
// the given op-name, args, and result, with deterministic timestamps.
func opIO(seq int, name string, args, result any) porcupine.Operation {
	return porcupine.Operation{
		Input:  model.OpInput{Name: name, Args: args},
		Call:   int64(seq * 2),
		Output: model.OpOutput{Result: result},
		Return: int64(seq*2 + 1),
	}
}

func TestCounter(t *testing.T) {
	t.Parallel()

	t.Run("Inc-then-Read at expected counts is linearizable", func(t *testing.T) {
		t.Parallel()
		m := linearize.Counter()
		history := []porcupine.Operation{
			opIO(0, "Inc", nil, int64(1)),
			opIO(1, "Inc", nil, int64(2)),
			opIO(2, "Read", nil, int64(2)),
		}
		testkit.True(t, porcupine.CheckOperations(m, history),
			"matching counter history should be linearizable")
	})

	t.Run("Read returning wrong value is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.Counter()
		history := []porcupine.Operation{
			opIO(0, "Inc", nil, int64(1)),
			opIO(1, "Read", nil, int64(99)),
		}
		testkit.False(t, porcupine.CheckOperations(m, history),
			"divergent Read should be rejected")
	})

	t.Run("Dec advances state correctly", func(t *testing.T) {
		t.Parallel()
		m := linearize.Counter()
		history := []porcupine.Operation{
			opIO(0, "Inc", int(5), int64(5)),
			opIO(1, "Dec", int(2), int64(3)),
			opIO(2, "Read", nil, int64(3)),
		}
		testkit.True(t, porcupine.CheckOperations(m, history),
			"Inc(5);Dec(2);Read=3 is linearizable")
	})

	t.Run("default delta is 1 when args is nil", func(t *testing.T) {
		t.Parallel()
		m := linearize.Counter()
		history := []porcupine.Operation{
			opIO(0, "Inc", nil, int64(1)),
			opIO(1, "Dec", nil, int64(0)),
		}
		testkit.True(t, porcupine.CheckOperations(m, history),
			"nil args defaults to delta=1")
	})

	t.Run("unknown op rejects", func(t *testing.T) {
		t.Parallel()
		m := linearize.Counter()
		history := []porcupine.Operation{
			opIO(0, "Mystery", nil, int64(0)),
		}
		testkit.False(t, porcupine.CheckOperations(m, history),
			"unknown op name should reject")
	})
}

// counterDelta and matchesInt64 absorb the several integer types a real
// counter might use for its arguments and returns. Both fall back to a
// defined behaviour on anything else rather than panicking, which is what
// these exercise.
func TestCounterModelTypeCoercion(t *testing.T) {
	t.Parallel()

	m := linearize.Counter()
	in := func(name string, args any) model.OpInput { return model.OpInput{Name: name, Args: args} }
	out := func(r any) model.OpOutput { return model.OpOutput{Result: r} }

	t.Run("Inc accepts int, int32 and int64 deltas", func(t *testing.T) {
		t.Parallel()
		for _, delta := range []any{int(3), int32(3), int64(3)} {
			ok, next := m.Step(m.Init(), in("Inc", delta), out(int64(3)))
			testkit.True(t, ok, "a delta of 3 must advance the counter to 3")
			testkit.Equal(t, next.(int64), int64(3), "state advances by the delta")
		}
	})

	// A nil or unrecognised argument means "step by one" rather than zero, so
	// an action that forgot to pass its delta still models a counter.
	t.Run("nil and unknown args default to a delta of one", func(t *testing.T) {
		t.Parallel()
		for _, args := range []any{nil, "not a number"} {
			ok, next := m.Step(m.Init(), in("Inc", args), out(int64(1)))
			testkit.True(t, ok, "an unusable delta defaults to 1")
			testkit.Equal(t, next.(int64), int64(1), "state advances by one")
		}
	})

	t.Run("Dec subtracts the delta", func(t *testing.T) {
		t.Parallel()
		ok, next := m.Step(m.Init(), in("Dec", 2), out(int64(-2)))
		testkit.True(t, ok, "Dec moves the counter down")
		testkit.Equal(t, next.(int64), int64(-2), "state decreases by the delta")
	})

	t.Run("results may be int, int32 or int64", func(t *testing.T) {
		t.Parallel()
		for _, result := range []any{int(1), int32(1), int64(1)} {
			ok, _ := m.Step(m.Init(), in("Inc", nil), out(result))
			testkit.True(t, ok, "any integer width may carry the result")
		}
	})

	t.Run("a non-integer result is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Read", nil), out("zero"))
		testkit.False(t, ok, "a non-numeric result cannot match a counter")
	})

	t.Run("an unknown operation is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Reset", nil), out(int64(0)))
		testkit.False(t, ok, "an unmodelled operation cannot be accepted")
	})

	t.Run("state helpers", func(t *testing.T) {
		t.Parallel()
		_, three := m.Step(m.Init(), in("Inc", 3), out(int64(3)))
		testkit.True(t, m.Equal(m.Init(), int64(0)), "zero equals a fresh counter")
		testkit.False(t, m.Equal(m.Init(), three), "different totals differ")
		testkit.Equal(t, m.DescribeState(three), "3", "renders the total")
		testkit.Contains(t, m.DescribeOperation(in("Inc", 1), out(int64(1))), "Inc", "names the operation")
	})
}
