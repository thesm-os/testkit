// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize_test

import (
	"testing"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/linearize"
)

func TestSet(t *testing.T) {
	t.Parallel()

	t.Run("Add-Contains-Remove cycle is linearizable", func(t *testing.T) {
		t.Parallel()
		m := linearize.Set()
		history := []porcupine.Operation{
			opCAS(0, "Add", "x", nil, false),
			opCAS(1, "Contains", "x", nil, true),
			opCAS(2, "Remove", "x", nil, true),
			opCAS(3, "Contains", "x", nil, false),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "set cycle")
	})

	t.Run("Add returning wrong prior is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.Set()
		history := []porcupine.Operation{
			opCAS(0, "Add", "x", nil, true), // claims already present, but empty
		}
		testkit.False(t, porcupine.CheckOperations(m, history), "Add reports wrong prior")
	})

	t.Run("Remove returning wrong prior is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.Set()
		history := []porcupine.Operation{
			opCAS(0, "Remove", "x", nil, true), // claims was present, but empty
		}
		testkit.False(t, porcupine.CheckOperations(m, history), "Remove reports wrong prior")
	})

	t.Run("Contains lying about membership is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.Set()
		history := []porcupine.Operation{
			opCAS(0, "Contains", "x", nil, true),
		}
		testkit.False(t, porcupine.CheckOperations(m, history), "Contains lies")
	})

	t.Run("operations across elements are independent", func(t *testing.T) {
		t.Parallel()
		m := linearize.Set()
		history := []porcupine.Operation{
			opCAS(0, "Add", "x", nil, false),
			opCAS(1, "Contains", "y", nil, false),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "partitions independent")
	})
}
