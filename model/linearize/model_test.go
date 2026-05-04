// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize_test

import (
	"errors"
	"testing"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit/model"
	"go.thesmos.sh/testkit/model/linearize"
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
