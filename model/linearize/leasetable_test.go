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
