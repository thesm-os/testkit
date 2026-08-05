// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize_test

import (
	"errors"
	"testing"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/linearize"
)

func TestAppendLog(t *testing.T) {
	t.Parallel()

	t.Run("Append-then-At-then-Len is linearizable", func(t *testing.T) {
		t.Parallel()
		m := linearize.AppendLog[string]()
		history := []porcupine.Operation{
			opIO(0, "Append", "a", int64(0)),
			opIO(1, "Append", "b", int64(1)),
			opIO(2, "At", int64(1), linearize.ReaderResult[string]{Value: "b"}),
			opIO(3, "Len", nil, int64(2)),
		}
		testkit.True(t, porcupine.CheckOperations(m, history),
			"matching append-log history should be linearizable")
	})

	t.Run("non-monotonic Append offset is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.AppendLog[string]()
		history := []porcupine.Operation{
			opIO(0, "Append", "a", int64(0)),
			opIO(1, "Append", "b", int64(99)),
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
			opIO(0, "Append", "a", int64(0)),
			opIO(1, "At", int64(0), linearize.ReaderResult[string]{Value: "wrong"}),
		}
		testkit.False(t, porcupine.CheckOperations(m, history),
			"wrong-value At should be rejected")
	})

	t.Run("Len with wrong count is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.AppendLog[string]()
		history := []porcupine.Operation{
			opIO(0, "Append", "a", int64(0)),
			opIO(1, "Len", nil, int64(99)),
		}
		testkit.False(t, porcupine.CheckOperations(m, history),
			"wrong Len should be rejected")
	})
}
