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

func TestNondeterministicModel(t *testing.T) {
	t.Parallel()

	t.Run("wrapped Counter accepts fault-marked silent-failure", func(t *testing.T) {
		t.Parallel()
		inner := linearize.Counter()
		nd := linearize.NondeterministicModel(inner)
		// Inc whose actual outcome is "no state change" — modeled by the
		// SUT returning the prior count. Under no fault, that would be
		// rejected. Under a fault marker on the input, it must pass.
		history := []porcupine.Operation{
			{
				Input: model.OpInput{
					Name: "Inc",
					Args: linearize.NondeterministicOutcome{Inner: nil},
				},
				Call:   0,
				Output: model.OpOutput{Result: int64(0)},
				Return: 1,
			},
		}
		testkit.True(t, porcupine.CheckOperations(nd, history),
			"fault-marked silent failure accepted")
	})

	t.Run("wrapped Counter accepts fault-marked success", func(t *testing.T) {
		t.Parallel()
		inner := linearize.Counter()
		nd := linearize.NondeterministicModel(inner)
		history := []porcupine.Operation{
			{
				Input: model.OpInput{
					Name: "Inc",
					Args: linearize.NondeterministicOutcome{Inner: nil},
				},
				Call:   0,
				Output: model.OpOutput{Result: int64(1)},
				Return: 1,
			},
		}
		testkit.True(t, porcupine.CheckOperations(nd, history),
			"fault-marked success accepted")
	})

	t.Run("unmarked input delegates to inner (strict)", func(t *testing.T) {
		t.Parallel()
		nd := linearize.NondeterministicModel(linearize.Counter())
		history := []porcupine.Operation{
			opIO(0, "Inc", nil, int64(0)), // Inc → 0 is wrong; inner must reject
		}
		testkit.False(t, porcupine.CheckOperations(nd, history),
			"unmarked input strict-checks via inner")
	})

	// Porcupine models may carry inputs of any shape. When the input is not
	// a testkit OpInput there is no fault marker to read, so the wrapper must
	// step aside rather than assert its way into a panic.
	t.Run("a foreign input shape delegates to inner", func(t *testing.T) {
		t.Parallel()
		var seen any
		inner := porcupine.Model{
			Init: func() any { return 0 },
			Step: func(state, input, _ any) (bool, any) {
				seen = input
				return true, state
			},
		}
		nd := linearize.NondeterministicModel(inner)

		ok, next := nd.Step(0, "raw-input", nil)
		testkit.True(t, ok, "the inner model's verdict stands")
		testkit.Equal(t, next, 0, "the inner model's state stands")
		testkit.Equal(t, seen, any("raw-input"), "the input reaches inner untouched")
	})
}
