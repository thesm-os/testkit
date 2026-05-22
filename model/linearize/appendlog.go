// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"fmt"

	"github.com/anishathalye/porcupine"
	"github.com/google/go-cmp/cmp"

	"go.thesmos.sh/testkit/model"
)

// AppendLog builds a porcupine.Model for append-only logs. State is
// a slice of E entries shared across the whole history. Step
// functions:
//
//   - Append: state = append(state, input). Output is the offset
//     (int64) of the newly appended entry; valid only if the offset
//     equals the prior length.
//   - At:     output equals state[input.(int64)] when in range; the
//     zero E with a non-nil error otherwise.
//   - Len:    output equals int64(len(state)); state unchanged.
//
// Operation names are fixed ("Append", "At", "Len"); the generator
// emits aliases when SUT method names differ.
func AppendLog[E any]() porcupine.Model {
	return porcupine.Model{
		Partition: singleHistoryPartition,
		Init:      func() any { return []E(nil) },
		Step: func(state, input, output any) (bool, any) {
			s := state.([]E)
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			switch inp.Name {
			case "Append":
				v, ok := inp.Args.(E)
				if !ok {
					return false, s
				}
				off, ok := out.Result.(int64)
				if !ok {
					return false, s
				}
				if off != int64(len(s)) {
					return false, s
				}
				next := make([]E, len(s)+1)
				copy(next, s)
				next[len(s)] = v
				return true, next
			case "At":
				i, ok := inp.Args.(int64)
				if !ok {
					return false, s
				}
				r, ok := out.Result.(ReaderResult[E])
				if !ok {
					return false, s
				}
				if i < 0 || i >= int64(len(s)) {
					return r.Err != nil, s
				}
				return r.Err == nil && cmp.Equal(r.Value, s[i]), s
			case "Len":
				n, ok := out.Result.(int64)
				if !ok {
					return false, s
				}
				return n == int64(len(s)), s
			default:
				return false, s
			}
		},
		Equal: func(a, b any) bool {
			return cmp.Equal(a, b)
		},
		DescribeOperation: func(input, output any) string {
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			return fmt.Sprintf("%s(%v) -> %v", inp.Name, inp.Args, out.Result)
		},
		DescribeState: func(state any) string {
			s := state.([]E)
			return fmt.Sprintf("len=%d", len(s))
		},
	}
}
