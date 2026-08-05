// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"errors"
	"fmt"
	"sort"

	"github.com/anishathalye/porcupine"
	"github.com/google/go-cmp/cmp"

	"go.thesmos.sh/testkit/engine/model"
)

// kvState is the per-partition state for a single key.
type kvState[V any] struct {
	present bool
	value   V
}

// KV builds a porcupine.Model for CRUD-shaped interfaces, partitioned
// by key. This is the model the generator emits for Reader/Writer/Deleter
// shapes.
//
// Per-partition state is {present, value} for a single key. Step
// functions:
//   - Get: valid if output matches state (value when present, sentinel when absent)
//   - Put: always valid, state becomes {true, input value}
//   - Delete: always valid, state becomes {false, zero}
//
// The sentinel error validates Get on absent keys. Pass nil if the
// interface doesn't use sentinel errors; the model then accepts any
// non-nil error as "key not present."
//
// The partition key comes from model.OpInput.PartitionKey, set by the
// ConcurrentAction helpers.
func KV[K comparable, V any](sentinel error) porcupine.Model {
	var zeroV V

	return porcupine.Model{
		Partition: func(history []porcupine.Operation) [][]porcupine.Operation {
			m := make(map[string][]porcupine.Operation)
			for _, op := range history {
				inp := op.Input.(model.OpInput)
				m[inp.PartitionKey] = append(m[inp.PartitionKey], op)
			}
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			// Stable order for deterministic checking.
			sort.Strings(keys)
			result := make([][]porcupine.Operation, 0, len(keys))
			for _, k := range keys {
				result = append(result, m[k])
			}
			return result
		},
		Init: func() any {
			return kvState[V]{present: false}
		},
		Step: func(state, input, output any) (bool, any) {
			s := state.(kvState[V])
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)

			switch inp.Name {
			case OpGet:
				r := out.Result.(ReaderResult[V])
				if !s.present {
					if sentinel == nil {
						return r.Err != nil, s
					}
					return errors.Is(r.Err, sentinel), s
				}
				return r.Err == nil && cmp.Equal(r.Value, s.value), s

			case OpPut:
				return true, kvState[V]{present: true, value: inp.Args.(V)}

			case OpDelete:
				return true, kvState[V]{present: false, value: zeroV}

			default:
				return false, s
			}
		},
		Equal: func(a, b any) bool {
			sa := a.(kvState[V])
			sb := b.(kvState[V])
			if sa.present != sb.present {
				return false
			}
			if !sa.present {
				return true
			}
			return cmp.Equal(sa.value, sb.value)
		},
		DescribeOperation: func(input, output any) string {
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			return fmt.Sprintf("%s(%v) -> %v", inp.Name, inp.Args, out.Result)
		},
		DescribeState: func(state any) string {
			s := state.(kvState[V])
			if !s.present {
				return "<absent>"
			}
			return fmt.Sprintf("%v", s.value)
		},
	}
}
