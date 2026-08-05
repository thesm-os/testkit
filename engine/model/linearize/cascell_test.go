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

	// A CAS carrying the current version is expected to win. Reporting an
	// error anyway is a defect the model has to reject, and it is distinct
	// from the version-mismatch case above it.
	t.Run("a CAS at the right version that errors is not linearizable", func(t *testing.T) {
		t.Parallel()
		m := linearize.CASCell[casEntry, int](notFound, mismatch, versionOf, nextVer)
		history := []porcupine.Operation{
			opCAS(0, "CAS", "k", casEntry{Version: 0, Value: "v"}, linearize.WriterResult{}),
			opCAS(1, "CAS", "k", casEntry{Version: 1, Value: "w"},
				linearize.WriterResult{Err: errors.New("write failed")}),
		}
		testkit.False(t, porcupine.CheckOperations(m, history),
			"a matching-version CAS must not fail")
	})
}

// The porcupine.Model fields are reachable directly, which is the only way to
// drive the defensive arms: a history that porcupine would accept never
// carries a mis-typed input or an unknown operation, but a hand-written model
// or a renamed action can produce exactly that. The model must reject rather
// than panic.
func TestCASCellModelDefensiveArms(t *testing.T) {
	t.Parallel()

	notFound := errors.New("not found")
	mismatch := errors.New("version mismatch")
	versionOf := func(e casEntry) int { return e.Version }
	nextVer := func(v int) int { return v + 1 }
	newModel := func() porcupine.Model {
		return linearize.CASCell[casEntry, int](notFound, mismatch, versionOf, nextVer)
	}
	in := func(name string, args any) model.OpInput {
		return model.OpInput{Name: name, Args: args, PartitionKey: "k"}
	}
	out := func(r any) model.OpOutput { return model.OpOutput{Result: r} }

	t.Run("Get with a non-reader result is rejected", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		ok, _ := m.Step(m.Init(), in("Get", nil), out("not a ReaderResult"))
		testkit.False(t, ok, "a mis-typed Get output cannot be linearizable")
	})

	t.Run("CAS with non-value args is rejected", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		ok, _ := m.Step(m.Init(), in("CAS", 12345), out(linearize.WriterResult{}))
		testkit.False(t, ok, "a mis-typed CAS argument cannot be linearizable")
	})

	t.Run("CAS with a non-writer result is rejected", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		ok, _ := m.Step(m.Init(), in("CAS", casEntry{}), out("not a WriterResult"))
		testkit.False(t, ok, "a mis-typed CAS output cannot be linearizable")
	})

	t.Run("an unknown operation is rejected", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		ok, _ := m.Step(m.Init(), in("Frobnicate", nil), out(nil))
		testkit.False(t, ok, "an operation the model does not model cannot be accepted")
	})

	t.Run("a failed first CAS leaves the cell absent", func(t *testing.T) {
		t.Parallel()
		m := newModel()
		ok, _ := m.Step(m.Init(), in("CAS", casEntry{Version: 0, Value: "v"}),
			out(linearize.WriterResult{Err: errors.New("boom")}))
		testkit.False(t, ok, "an errored write into an empty cell is not a valid transition")
	})

	// With no sentinel configured the model cannot check which error a read of
	// an absent key returned, only that it returned one.
	t.Run("a nil sentinel accepts any error on an absent key", func(t *testing.T) {
		t.Parallel()
		m := linearize.CASCell[casEntry, int](nil, mismatch, versionOf, nextVer)
		ok, _ := m.Step(m.Init(), in("Get", nil),
			out(linearize.ReaderResult[casEntry]{Err: errors.New("anything")}))
		testkit.True(t, ok, "any error is acceptable when no sentinel is configured")

		ok, _ = m.Step(m.Init(), in("Get", nil), out(linearize.ReaderResult[casEntry]{}))
		testkit.False(t, ok, "a successful read of an absent key is still wrong")
	})
}

func TestCASCellModelStateHelpers(t *testing.T) {
	t.Parallel()

	notFound := errors.New("not found")
	mismatch := errors.New("version mismatch")
	m := linearize.CASCell[casEntry, int](notFound, mismatch,
		func(e casEntry) int { return e.Version },
		func(v int) int { return v + 1 })
	in := func(name string, args any) model.OpInput {
		return model.OpInput{Name: name, Args: args, PartitionKey: "k"}
	}
	out := func(r any) model.OpOutput { return model.OpOutput{Result: r} }

	empty := m.Init()
	_, filled := m.Step(empty, in("CAS", casEntry{Version: 0, Value: "v"}), out(linearize.WriterResult{}))
	_, other := m.Step(empty, in("CAS", casEntry{Version: 0, Value: "w"}), out(linearize.WriterResult{}))

	t.Run("Equal distinguishes presence and value", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, m.Equal(empty, m.Init()), "two absent cells are equal")
		testkit.False(t, m.Equal(empty, filled), "absent and present differ")
		testkit.True(t, m.Equal(filled, filled), "a state equals itself")
		testkit.False(t, m.Equal(filled, other), "same version, different value")
	})

	// DescribeState and DescribeOperation feed the failure visualisation, so
	// they must render both the absent and present cases legibly.
	t.Run("DescribeState renders both cases", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, m.DescribeState(empty), "<absent>", "absent cell")
		testkit.Contains(t, m.DescribeState(filled), "v=", "present cell shows its value")
	})

	t.Run("DescribeOperation renders call and result", func(t *testing.T) {
		t.Parallel()
		got := m.DescribeOperation(in("CAS", casEntry{Version: 1, Value: "x"}), out(linearize.WriterResult{}))
		testkit.Contains(t, got, "CAS", "names the operation")
		testkit.Contains(t, got, "->", "separates call from result")
	})
}
