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

// Add and Remove return the *prior* membership, so a set that reports the new
// state instead is a real defect the model must catch. Contains returns the
// current state, which is why it is exempt from the shared bool decode.
func TestSetModelBranches(t *testing.T) {
	t.Parallel()

	m := linearize.Set()
	in := func(name string) model.OpInput { return model.OpInput{Name: name, PartitionKey: "k"} }
	out := func(r any) model.OpOutput { return model.OpOutput{Result: r} }

	t.Run("Add reports the prior absence and marks present", func(t *testing.T) {
		t.Parallel()
		ok, next := m.Step(m.Init(), in("Add"), out(false))
		testkit.True(t, ok, "adding to an empty set reports prior=false")
		ok, _ = m.Step(next, in("Contains"), out(true))
		testkit.True(t, ok, "the element is present afterwards")
	})

	t.Run("Add reporting the wrong prior is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Add"), out(true))
		testkit.False(t, ok, "an empty set cannot report a prior presence")
	})

	t.Run("Remove reports the prior presence and clears it", func(t *testing.T) {
		t.Parallel()
		_, present := m.Step(m.Init(), in("Add"), out(false))
		ok, cleared := m.Step(present, in("Remove"), out(true))
		testkit.True(t, ok, "removing a present element reports prior=true")
		ok, _ = m.Step(cleared, in("Contains"), out(false))
		testkit.True(t, ok, "the element is absent afterwards")
	})

	t.Run("Remove reporting the wrong prior is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Remove"), out(true))
		testkit.False(t, ok, "an empty set cannot report a prior presence")
	})

	t.Run("Contains reporting the wrong membership is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Contains"), out(true))
		testkit.False(t, ok, "an empty set does not contain the element")
	})

	t.Run("a non-bool result is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Add"), out("yes"))
		testkit.False(t, ok, "membership results must be boolean")

		ok, _ = m.Step(m.Init(), in("Contains"), out("yes"))
		testkit.False(t, ok, "Contains decodes its own result and rejects non-bools")
	})

	t.Run("an unknown operation is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Toggle"), out(true))
		testkit.False(t, ok, "an unmodelled operation cannot be accepted")
	})

	t.Run("state helpers", func(t *testing.T) {
		t.Parallel()
		absent := m.Init()
		_, present := m.Step(absent, in("Add"), out(false))
		testkit.True(t, m.Equal(absent, m.Init()), "two empty sets are equal")
		testkit.False(t, m.Equal(absent, present), "membership is the state")
		testkit.Equal(t, m.DescribeState(absent), "<out>", "absent")
		testkit.Equal(t, m.DescribeState(present), "<in>", "present")
		testkit.Contains(t, m.DescribeOperation(in("Add"), out(false)), "Add", "names the operation")
	})
}
