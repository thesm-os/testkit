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

func TestPool(t *testing.T) {
	t.Parallel()

	t.Run("balanced Get-Put cycle is linearizable", func(t *testing.T) {
		t.Parallel()
		m := linearize.Pool()
		history := []porcupine.Operation{
			opIO(0, "Get", nil, linearize.WriterResult{}),
			opIO(1, "Put", nil, linearize.WriterResult{}),
			opIO(2, "Get", nil, linearize.WriterResult{}),
			opIO(3, "Put", nil, linearize.WriterResult{}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "balanced cycle")
	})

	t.Run("Get returning an error is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.Pool()
		history := []porcupine.Operation{
			opIO(0, "Get", nil, linearize.WriterResult{Err: errors.New("nope")}),
		}
		testkit.False(t, porcupine.CheckOperations(m, history), "Get must not error")
	})

	t.Run("double Put surfaces an error", func(t *testing.T) {
		t.Parallel()
		m := linearize.Pool()
		history := []porcupine.Operation{
			opIO(0, "Get", nil, linearize.WriterResult{}),
			opIO(1, "Put", nil, linearize.WriterResult{}),
			opIO(2, "Put", nil, linearize.WriterResult{Err: errors.New("double-put")}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "double Put returns error")
	})

	t.Run("double Put swallowing error is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.Pool()
		history := []porcupine.Operation{
			opIO(0, "Get", nil, linearize.WriterResult{}),
			opIO(1, "Put", nil, linearize.WriterResult{}),
			opIO(2, "Put", nil, linearize.WriterResult{}), // no error
		}
		testkit.False(t, porcupine.CheckOperations(m, history), "double Put must error")
	})
}

// A pool tracks outstanding checkouts: Put without a matching Get is the
// classic double-release bug, and the model must require it to error rather
// than silently drive the count negative.
func TestPoolModelBranches(t *testing.T) {
	t.Parallel()

	m := linearize.Pool()
	in := func(name string) model.OpInput { return model.OpInput{Name: name} }
	out := func(r any) model.OpOutput { return model.OpOutput{Result: r} }

	t.Run("get then put balances", func(t *testing.T) {
		t.Parallel()
		ok, got := m.Step(m.Init(), in("Get"), out(linearize.WriterResult{}))
		testkit.True(t, ok, "a Get from the pool succeeds")
		ok, _ = m.Step(got, in("Put"), out(linearize.WriterResult{}))
		testkit.True(t, ok, "returning a checked-out item succeeds")
	})

	t.Run("a failing Get is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Get"), out(linearize.WriterResult{Err: errors.New("empty")}))
		testkit.False(t, ok, "this model treats Get as always available")
	})

	t.Run("Put without a matching Get must error", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Put"), out(linearize.WriterResult{Err: errors.New("not checked out")}))
		testkit.True(t, ok, "an unmatched Put reports an error")
		ok, _ = m.Step(m.Init(), in("Put"), out(linearize.WriterResult{}))
		testkit.False(t, ok, "an unmatched Put must not succeed")
	})

	t.Run("a failing Put of a checked-out item is rejected", func(t *testing.T) {
		t.Parallel()
		_, got := m.Step(m.Init(), in("Get"), out(linearize.WriterResult{}))
		ok, _ := m.Step(got, in("Put"), out(linearize.WriterResult{Err: errors.New("io")}))
		testkit.False(t, ok, "returning a held item must succeed")
	})

	t.Run("a non-writer result is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Get"), out("nonsense"))
		testkit.False(t, ok, "pool results must be WriterResults")
	})

	t.Run("an unknown operation is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Drain"), out(linearize.WriterResult{}))
		testkit.False(t, ok, "an unmodelled operation cannot be accepted")
	})

	t.Run("state helpers track outstanding checkouts", func(t *testing.T) {
		t.Parallel()
		empty := m.Init()
		_, one := m.Step(empty, in("Get"), out(linearize.WriterResult{}))
		testkit.True(t, m.Equal(empty, m.Init()), "two idle pools are equal")
		testkit.False(t, m.Equal(empty, one), "outstanding count is part of the state")
		testkit.Contains(t, m.DescribeState(one), "outstanding=1", "renders the checkout count")
		testkit.Contains(t, m.DescribeOperation(in("Get"), out(nil)), "Get", "names the operation")
	})
}
