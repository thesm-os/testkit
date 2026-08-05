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
