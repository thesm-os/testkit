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
