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

type casEntry struct {
	Version int
	Value   string
}

// opCAS wraps an op input with a partition key (CASCell partitions by key).
func opCAS(seq int, name, key string, args, result any) porcupine.Operation {
	return porcupine.Operation{
		Input:  model.OpInput{Name: name, Args: args, PartitionKey: key},
		Call:   int64(seq * 2),
		Output: model.OpOutput{Result: result},
		Return: int64(seq*2 + 1),
	}
}

func TestCASCell(t *testing.T) {
	t.Parallel()

	notFound := errors.New("not found")
	mismatch := errors.New("version mismatch")

	versionOf := func(e casEntry) int { return e.Version }
	nextVer := func(v int) int { return v + 1 }

	t.Run("first CAS into empty cell succeeds", func(t *testing.T) {
		t.Parallel()
		m := linearize.CASCell[casEntry, int](notFound, mismatch, versionOf, nextVer)
		history := []porcupine.Operation{
			opCAS(0, "CAS", "k", casEntry{Version: 0, Value: "v"}, linearize.WriterResult{}),
			opCAS(1, "Get", "k", nil, linearize.ReaderResult[casEntry]{Value: casEntry{Version: 0, Value: "v"}}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "empty-cell first write")
	})

	t.Run("Get on empty key returns sentinel", func(t *testing.T) {
		t.Parallel()
		m := linearize.CASCell[casEntry, int](notFound, mismatch, versionOf, nextVer)
		history := []porcupine.Operation{
			opCAS(0, "Get", "k", nil, linearize.ReaderResult[casEntry]{Err: notFound}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "absent key returns sentinel")
	})

	t.Run("CAS with wrong version yields mismatch error", func(t *testing.T) {
		t.Parallel()
		m := linearize.CASCell[casEntry, int](notFound, mismatch, versionOf, nextVer)
		history := []porcupine.Operation{
			opCAS(0, "CAS", "k", casEntry{Version: 0, Value: "v1"}, linearize.WriterResult{}),
			opCAS(1, "CAS", "k", casEntry{Version: 99, Value: "v2"}, linearize.WriterResult{Err: mismatch}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "version mismatch rejected")
	})

	t.Run("CAS swallowing the mismatch error is rejected", func(t *testing.T) {
		t.Parallel()
		m := linearize.CASCell[casEntry, int](notFound, mismatch, versionOf, nextVer)
		history := []porcupine.Operation{
			opCAS(0, "CAS", "k", casEntry{Version: 0, Value: "v1"}, linearize.WriterResult{}),
			opCAS(1, "CAS", "k", casEntry{Version: 99, Value: "v2"}, linearize.WriterResult{}), // no error
		}
		testkit.False(t, porcupine.CheckOperations(m, history), "mismatch must surface")
	})

	t.Run("two writes share a key only on matching versions", func(t *testing.T) {
		t.Parallel()
		m := linearize.CASCell[casEntry, int](notFound, mismatch, versionOf, nextVer)
		// nextVer advances stored version by +1 after each successful write.
		history := []porcupine.Operation{
			opCAS(0, "CAS", "k", casEntry{Version: 0, Value: "a"}, linearize.WriterResult{}),
			opCAS(1, "CAS", "k", casEntry{Version: 1, Value: "b"}, linearize.WriterResult{}),
			opCAS(2, "Get", "k", nil, linearize.ReaderResult[casEntry]{Value: casEntry{Version: 1, Value: "b"}}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "version-chained writes")
	})

	t.Run("operations across keys are independent", func(t *testing.T) {
		t.Parallel()
		m := linearize.CASCell[casEntry, int](notFound, mismatch, versionOf, nextVer)
		history := []porcupine.Operation{
			opCAS(0, "CAS", "k1", casEntry{Version: 0, Value: "a"}, linearize.WriterResult{}),
			opCAS(1, "CAS", "k2", casEntry{Version: 0, Value: "b"}, linearize.WriterResult{}),
			opCAS(2, "Get", "k1", nil, linearize.ReaderResult[casEntry]{Value: casEntry{Version: 0, Value: "a"}}),
			opCAS(3, "Get", "k2", nil, linearize.ReaderResult[casEntry]{Value: casEntry{Version: 0, Value: "b"}}),
		}
		testkit.True(t, porcupine.CheckOperations(m, history), "independent partitions")
	})
}
