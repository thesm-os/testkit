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

var errNotFound = errors.New("not found")

func TestKVModelLinearizable(t *testing.T) {
	t.Parallel()
	m := linearize.KV[string, string](errNotFound)

	// Linear history: put("a", "x"), get("a") -> "x"
	history := []porcupine.Operation{
		{
			ClientId: 0, Call: 1, Return: 2,
			Input:  model.OpInput{Name: "Put", PartitionKey: "a", Args: "x"},
			Output: model.OpOutput{Result: linearize.WriterResult{}},
		},
		{
			ClientId: 0, Call: 3, Return: 4,
			Input:  model.OpInput{Name: "Get", PartitionKey: "a", Args: "a"},
			Output: model.OpOutput{Result: linearize.ReaderResult[string]{Value: "x"}},
		},
	}
	if !porcupine.CheckOperations(m, history) {
		t.Fatal("linear put-then-get must be linearizable")
	}
}

func TestKVModelNotLinearizable(t *testing.T) {
	t.Parallel()
	m := linearize.KV[string, string](errNotFound)

	// Impossible: get("a") returns "x" but no put happened.
	history := []porcupine.Operation{
		{
			ClientId: 0, Call: 1, Return: 2,
			Input:  model.OpInput{Name: "Get", PartitionKey: "a", Args: "a"},
			Output: model.OpOutput{Result: linearize.ReaderResult[string]{Value: "x"}},
		},
	}
	if porcupine.CheckOperations(m, history) {
		t.Fatal("get returning value with no prior put must NOT be linearizable")
	}
}

func TestKVModelSentinel(t *testing.T) {
	t.Parallel()
	m := linearize.KV[string, string](errNotFound)

	// get("a") returns sentinel error — valid for empty store.
	history := []porcupine.Operation{
		{
			ClientId: 0, Call: 1, Return: 2,
			Input:  model.OpInput{Name: "Get", PartitionKey: "a", Args: "a"},
			Output: model.OpOutput{Result: linearize.ReaderResult[string]{Err: errNotFound}},
		},
	}
	if !porcupine.CheckOperations(m, history) {
		t.Fatal("get returning sentinel on empty store must be linearizable")
	}
}

func TestKVModelDelete(t *testing.T) {
	t.Parallel()
	m := linearize.KV[string, string](errNotFound)

	// put("a", "x"), delete("a"), get("a") -> sentinel
	history := []porcupine.Operation{
		{
			ClientId: 0, Call: 1, Return: 2,
			Input:  model.OpInput{Name: "Put", PartitionKey: "a", Args: "x"},
			Output: model.OpOutput{Result: linearize.WriterResult{}},
		},
		{
			ClientId: 0, Call: 3, Return: 4,
			Input:  model.OpInput{Name: "Delete", PartitionKey: "a", Args: "a"},
			Output: model.OpOutput{Result: linearize.WriterResult{}},
		},
		{
			ClientId: 0, Call: 5, Return: 6,
			Input:  model.OpInput{Name: "Get", PartitionKey: "a", Args: "a"},
			Output: model.OpOutput{Result: linearize.ReaderResult[string]{Err: errNotFound}},
		},
	}
	if !porcupine.CheckOperations(m, history) {
		t.Fatal("get after delete must return sentinel")
	}
}

func TestKVModelConcurrentLinearizable(t *testing.T) {
	t.Parallel()
	m := linearize.KV[string, string](errNotFound)

	// Two concurrent puts to different keys — always linearizable.
	history := []porcupine.Operation{
		{
			ClientId: 0, Call: 1, Return: 3,
			Input:  model.OpInput{Name: "Put", PartitionKey: "a", Args: "x"},
			Output: model.OpOutput{Result: linearize.WriterResult{}},
		},
		{
			ClientId: 1, Call: 2, Return: 4,
			Input:  model.OpInput{Name: "Put", PartitionKey: "b", Args: "y"},
			Output: model.OpOutput{Result: linearize.WriterResult{}},
		},
	}
	if !porcupine.CheckOperations(m, history) {
		t.Fatal("concurrent puts to different keys must be linearizable")
	}
}

func TestKVModelNilSentinel(t *testing.T) {
	t.Parallel()
	// nil sentinel: any non-nil error accepted as "absent."
	m := linearize.KV[string, string](nil)

	history := []porcupine.Operation{
		{
			ClientId: 0, Call: 1, Return: 2,
			Input:  model.OpInput{Name: "Get", PartitionKey: "a", Args: "a"},
			Output: model.OpOutput{Result: linearize.ReaderResult[string]{Err: errors.New("custom not found")}},
		},
	}
	if !porcupine.CheckOperations(m, history) {
		t.Fatal("any non-nil error should be accepted as absent when sentinel is nil")
	}
}

func TestModelBuilderCustomState(t *testing.T) {
	t.Parallel()

	// Custom state: counter that increments on "Inc" and reads on "Read".
	type counterState struct{ n int }

	m := linearize.NewModelBuilder(counterState{0}, nil).
		AddOp(linearize.OpSpec{
			Name: "Inc",
			Step: func(state, _, _ any) (bool, any) {
				s := state.(counterState)
				return true, counterState{n: s.n + 1}
			},
		}).
		AddOp(linearize.OpSpec{
			Name: "Read",
			Step: func(state, _, output any) (bool, any) {
				s := state.(counterState)
				out := output.(int)
				return out == s.n, state
			},
		}).
		Build()

	// Two increments then read -> 2: linearizable.
	history := []porcupine.Operation{
		{
			ClientId: 0, Call: 1, Return: 2,
			Input: model.OpInput{Name: "Inc"}, Output: model.OpOutput{},
		},
		{
			ClientId: 0, Call: 3, Return: 4,
			Input: model.OpInput{Name: "Inc"}, Output: model.OpOutput{},
		},
		{
			ClientId: 0, Call: 5, Return: 6,
			Input: model.OpInput{Name: "Read"}, Output: model.OpOutput{Result: 2},
		},
	}
	if !porcupine.CheckOperations(m, history) {
		t.Fatal("counter Inc+Inc+Read(2) must be linearizable")
	}

	// Read -> 1 with no Inc: not linearizable.
	bad := []porcupine.Operation{
		{
			ClientId: 0, Call: 1, Return: 2,
			Input: model.OpInput{Name: "Read"}, Output: model.OpOutput{Result: 1},
		},
	}
	if porcupine.CheckOperations(m, bad) {
		t.Fatal("Read(1) with no Inc must NOT be linearizable")
	}
}

// ModelBuilder assembles a porcupine.Model from registered op specs. The
// interesting behaviour is what it does with an operation nobody registered,
// how it partitions when only some ops declare a key, and the default
// equality it substitutes when the caller supplies none.
func TestModelBuilderAssembly(t *testing.T) {
	t.Parallel()

	type counterState struct{ N int }

	newBuilt := func(equal func(counterState, counterState) bool) porcupine.Model {
		return linearize.NewModelBuilder(counterState{}, equal).
			AddOp(linearize.OpSpec{
				Name: "Inc",
				Step: func(state, _, _ any) (bool, any) {
					s := state.(counterState)
					return true, counterState{N: s.N + 1}
				},
				PartitionKey: func(args any) string { return args.(string) },
			}).
			AddOp(linearize.OpSpec{
				Name: "Peek",
				Step: func(state, _, result any) (bool, any) {
					return result.(int) == state.(counterState).N, state
				},
			}).
			Build()
	}

	in := func(name string, args any) model.OpInput { return model.OpInput{Name: name, Args: args} }
	out := func(r any) model.OpOutput { return model.OpOutput{Result: r} }

	t.Run("a registered op runs its Step", func(t *testing.T) {
		t.Parallel()
		m := newBuilt(nil)
		ok, next := m.Step(m.Init(), in("Inc", "k"), out(nil))
		testkit.True(t, ok, "the registered step accepted the operation")
		testkit.Equal(t, next.(counterState).N, 1, "the step advanced the state")
	})

	t.Run("an unregistered op is rejected without touching state", func(t *testing.T) {
		t.Parallel()
		m := newBuilt(nil)
		ok, next := m.Step(m.Init(), in("Unknown", "k"), out(nil))
		testkit.False(t, ok, "an operation with no registered step cannot be accepted")
		testkit.Equal(t, next.(counterState).N, 0, "state is returned unchanged")
	})

	// Ops without a PartitionKey all land in the empty-string partition, so a
	// history mixing keyed and unkeyed ops splits into one partition per key
	// plus one shared partition.
	t.Run("partitioning splits by key and groups unkeyed ops", func(t *testing.T) {
		t.Parallel()
		m := newBuilt(nil)
		history := []porcupine.Operation{
			{Input: in("Inc", "a"), Output: out(nil), Call: 0, Return: 1},
			{Input: in("Inc", "b"), Output: out(nil), Call: 2, Return: 3},
			{Input: in("Peek", nil), Output: out(0), Call: 4, Return: 5},
		}
		parts := m.Partition(history)
		testkit.Equal(t, len(parts), 3, "two keys plus the unkeyed group")
	})

	t.Run("the default equality compares states structurally", func(t *testing.T) {
		t.Parallel()
		m := newBuilt(nil)
		testkit.True(t, m.Equal(counterState{N: 1}, counterState{N: 1}), "equal states")
		testkit.False(t, m.Equal(counterState{N: 1}, counterState{N: 2}), "different states")
	})

	t.Run("a supplied equality replaces the default", func(t *testing.T) {
		t.Parallel()
		// Deliberately coarse: every state is equivalent.
		m := newBuilt(func(counterState, counterState) bool { return true })
		testkit.True(t, m.Equal(counterState{N: 1}, counterState{N: 99}),
			"the supplied comparison wins over structural equality")
	})

	t.Run("describe helpers render operation and state", func(t *testing.T) {
		t.Parallel()
		m := newBuilt(nil)
		testkit.Contains(t, m.DescribeOperation(in("Inc", "k"), out(nil)), "Inc", "names the operation")
		testkit.Contains(t, m.DescribeState(counterState{N: 3}), "3", "renders the state")
	})
}

// An operation name the model does not know cannot be modelled, so the step
// must be rejected rather than silently treated as a no-op that linearizes.
func TestKVModelRejectsUnknownOperation(t *testing.T) {
	t.Parallel()
	m := linearize.KV[string, string](errNotFound)

	history := []porcupine.Operation{
		{
			ClientId: 0, Call: 1, Return: 2,
			Input:  model.OpInput{Name: "Truncate", PartitionKey: "a", Args: "a"},
			Output: model.OpOutput{Result: linearize.WriterResult{}},
		},
	}
	if porcupine.CheckOperations(m, history) {
		t.Fatal("an unmodelled operation must not be accepted")
	}
}
