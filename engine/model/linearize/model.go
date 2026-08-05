// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"fmt"
	"sort"

	"github.com/anishathalye/porcupine"
	"github.com/google/go-cmp/cmp"

	"go.thesmos.sh/testkit/engine/model"
)

// ModelBuilder assembles a porcupine.Model with typed state S.
// Consumer defines state type, initial state, equality, and per-op
// Step functions. The builder handles all untyped Porcupine wrapping.
type ModelBuilder[S any] struct {
	init  S
	equal func(S, S) bool
	ops   []OpSpec
}

// NewModelBuilder creates a builder with the given initial per-partition
// state and equality function. If equal is nil, cmp.Equal is used.
func NewModelBuilder[S any](init S, equal func(S, S) bool) *ModelBuilder[S] {
	return &ModelBuilder[S]{init: init, equal: equal}
}

// AddOp registers an operation spec.
func (b *ModelBuilder[S]) AddOp(spec OpSpec) *ModelBuilder[S] {
	b.ops = append(b.ops, spec)
	return b
}

// Build constructs a porcupine.Model from the registered operations.
func (b *ModelBuilder[S]) Build() porcupine.Model {
	stepByName := make(map[string]func(any, any, any) (bool, any))
	partKeyByName := make(map[string]func(any) string)
	for _, op := range b.ops {
		stepByName[op.Name] = op.Step
		if op.PartitionKey != nil {
			partKeyByName[op.Name] = op.PartitionKey
		}
	}

	equal := b.equal
	if equal == nil {
		equal = func(a, b S) bool { return cmp.Equal(a, b) }
	}

	return porcupine.Model{
		Partition: func(history []porcupine.Operation) [][]porcupine.Operation {
			m := make(map[string][]porcupine.Operation)
			for _, op := range history {
				inp := op.Input.(model.OpInput)
				key := ""
				if fn, ok := partKeyByName[inp.Name]; ok {
					key = fn(inp.Args)
				}
				m[key] = append(m[key], op)
			}
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			result := make([][]porcupine.Operation, 0, len(keys))
			for _, k := range keys {
				result = append(result, m[k])
			}
			return result
		},
		Init: func() any {
			return b.init
		},
		Step: func(state, input, output any) (bool, any) {
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			stepFn, ok := stepByName[inp.Name]
			if !ok {
				return false, state
			}
			return stepFn(state, inp.Args, out.Result)
		},
		Equal: func(a, b any) bool {
			return equal(a.(S), b.(S))
		},
		DescribeOperation: func(input, output any) string {
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			return fmt.Sprintf("%s(%v) -> %v", inp.Name, inp.Args, out.Result)
		},
		DescribeState: func(state any) string {
			return fmt.Sprintf("%v", state)
		},
	}
}
