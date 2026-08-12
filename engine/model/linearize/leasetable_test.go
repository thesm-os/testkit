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

func TestLeaseTable(t *testing.T) {
	t.Parallel()

	held := errors.New("held")
	free := errors.New("free")

	t.Run("Acquire-Release-Acquire cycle is linearizable", func(t *testing.T) {
		t.Parallel()
		m := linearize.LeaseTable(held, free)
		history := []porcupine.Operation{
			opCAS(0, "Acquire", "k", nil, linearize.WriterResult{}),
			opCAS(1, "Release", "k", nil, linearize.WriterResult{}),
			opCAS(2, "Acquire", "k", nil, linearize.WriterResult{}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "cycle holds")
	})

	t.Run("double Acquire surfaces held error", func(t *testing.T) {
		t.Parallel()
		m := linearize.LeaseTable(held, free)
		history := []porcupine.Operation{
			opCAS(0, "Acquire", "k", nil, linearize.WriterResult{}),
			opCAS(1, "Acquire", "k", nil, linearize.WriterResult{Err: held}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "second acquire returns held")
	})

	t.Run("Acquire swallowing held error is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.LeaseTable(held, free)
		history := []porcupine.Operation{
			opCAS(0, "Acquire", "k", nil, linearize.WriterResult{}),
			opCAS(1, "Acquire", "k", nil, linearize.WriterResult{}), // no error
		}
		testkit.False(t, porcupine.CheckOperations(m, history), "must surface held")
	})

	t.Run("Release of unheld surfaces free error", func(t *testing.T) {
		t.Parallel()
		m := linearize.LeaseTable(held, free)
		history := []porcupine.Operation{
			opCAS(0, "Release", "k", nil, linearize.WriterResult{Err: free}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "unheld release returns free")
	})

	t.Run("Release of unheld swallowing free error is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.LeaseTable(held, free)
		history := []porcupine.Operation{
			opCAS(0, "Release", "k", nil, linearize.WriterResult{}),
		}
		testkit.False(t, porcupine.CheckOperations(m, history), "must surface free")
	})

	t.Run("a nil freeErr speaks the lenient dialect", func(t *testing.T) {
		t.Parallel()
		// Giving up what was never taken is ordinary Go to a deferred
		// caller; with no strict sentinel named, silence is the answer and
		// an error is the violation.
		m := linearize.LeaseTable(held, nil)
		silent := []porcupine.Operation{
			opCAS(0, "Release", "k", nil, linearize.WriterResult{}),
		}
		testkit.True(t, porcupine.CheckOperations(m, silent), "a silent unheld release linearizes")
		refused := []porcupine.Operation{
			opCAS(0, "Release", "k", nil, linearize.WriterResult{Err: free}),
		}
		testkit.False(t, porcupine.CheckOperations(m, refused),
			"a refusing one claims a dialect nothing named")
	})

	t.Run("nil sentinel accepts any non-nil error", func(t *testing.T) {
		t.Parallel()
		m := linearize.LeaseTable(nil, nil)
		history := []porcupine.Operation{
			opCAS(0, "Acquire", "k", nil, linearize.WriterResult{}),
			opCAS(1, "Acquire", "k", nil, linearize.WriterResult{Err: errors.New("anything")}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "nil sentinel accepts arbitrary error")
	})

	t.Run("separate keys are independent", func(t *testing.T) {
		t.Parallel()
		m := linearize.LeaseTable(held, free)
		history := []porcupine.Operation{
			opCAS(0, "Acquire", "k1", nil, linearize.WriterResult{}),
			opCAS(1, "Acquire", "k2", nil, linearize.WriterResult{}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "independent partitions")
	})
}

// A lease is a mutex with an identity: acquiring a held lease must fail with
// the held sentinel, releasing a free one with the free sentinel. Returning
// success in either case is the defect this catches.
func TestLeaseTableModelBranches(t *testing.T) {
	t.Parallel()

	held := errors.New("already held")
	free := errors.New("not held")
	m := linearize.LeaseTable(held, free)
	in := func(name string) model.OpInput { return model.OpInput{Name: name, PartitionKey: "k"} }
	out := func(r any) model.OpOutput { return model.OpOutput{Result: r} }

	t.Run("acquire then release round-trips", func(t *testing.T) {
		t.Parallel()
		ok, acquired := m.Step(m.Init(), in("Acquire"), out(linearize.WriterResult{}))
		testkit.True(t, ok, "acquiring a free lease succeeds")
		ok, _ = m.Step(acquired, in("Release"), out(linearize.WriterResult{}))
		testkit.True(t, ok, "releasing a held lease succeeds")
	})

	t.Run("double acquire must surface the held sentinel", func(t *testing.T) {
		t.Parallel()
		_, acquired := m.Step(m.Init(), in("Acquire"), out(linearize.WriterResult{}))
		ok, _ := m.Step(acquired, in("Acquire"), out(linearize.WriterResult{Err: held}))
		testkit.True(t, ok, "the second acquire reports the held sentinel")
		ok, _ = m.Step(acquired, in("Acquire"), out(linearize.WriterResult{}))
		testkit.False(t, ok, "a lease cannot be acquired twice successfully")
	})

	t.Run("release of a free lease must surface the free sentinel", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Release"), out(linearize.WriterResult{Err: free}))
		testkit.True(t, ok, "releasing a free lease reports the free sentinel")
		ok, _ = m.Step(m.Init(), in("Release"), out(linearize.WriterResult{}))
		testkit.False(t, ok, "releasing a lease nobody holds cannot succeed")
	})

	t.Run("a failing acquire on a free lease is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Acquire"), out(linearize.WriterResult{Err: errors.New("io")}))
		testkit.False(t, ok, "a free lease must be acquirable")
	})

	t.Run("a failing release on a held lease is rejected", func(t *testing.T) {
		t.Parallel()
		_, acquired := m.Step(m.Init(), in("Acquire"), out(linearize.WriterResult{}))
		ok, _ := m.Step(acquired, in("Release"), out(linearize.WriterResult{Err: errors.New("io")}))
		testkit.False(t, ok, "a held lease must be releasable")
	})

	t.Run("a non-writer result is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Acquire"), out("nonsense"))
		testkit.False(t, ok, "lease results must be WriterResults")
	})

	t.Run("an unknown operation is rejected", func(t *testing.T) {
		t.Parallel()
		ok, _ := m.Step(m.Init(), in("Renew"), out(linearize.WriterResult{}))
		testkit.False(t, ok, "an unmodelled operation cannot be accepted")
	})

	t.Run("state helpers", func(t *testing.T) {
		t.Parallel()
		freeState := m.Init()
		_, heldState := m.Step(freeState, in("Acquire"), out(linearize.WriterResult{}))
		testkit.True(t, m.Equal(freeState, m.Init()), "two free leases are equal")
		testkit.False(t, m.Equal(freeState, heldState), "held-ness is the state")
		testkit.Equal(t, m.DescribeState(freeState), "<free>", "free")
		testkit.Equal(t, m.DescribeState(heldState), "<held>", "held")
		testkit.Contains(t, m.DescribeOperation(in("Acquire"), out(nil)), "Acquire", "names the operation")
	})
}
