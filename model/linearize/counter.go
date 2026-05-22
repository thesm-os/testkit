// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"fmt"
	"strconv"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit/model"
)

// Counter builds a porcupine.Model for monotonically-mutated integer
// counters. State is a single int shared across the whole history
// (no partitioning). Step functions:
//
//   - Inc:  state += input delta (typically 1); output is the new value
//   - Dec:  state -= input delta; output is the new value
//   - Read: output equals state; state unchanged
//
// Operation names are fixed ("Inc", "Dec", "Read"). The generator
// emits aliases when the SUT method names differ.
func Counter() porcupine.Model {
	return porcupine.Model{
		Partition: singleHistoryPartition,
		Init:      func() any { return int64(0) },
		Step: func(state, input, output any) (bool, any) {
			s := state.(int64)
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			switch inp.Name {
			case "Inc":
				delta := counterDelta(inp.Args)
				next := s + delta
				return matchesInt64(out.Result, next), next
			case "Dec":
				delta := counterDelta(inp.Args)
				next := s - delta
				return matchesInt64(out.Result, next), next
			case "Read":
				return matchesInt64(out.Result, s), s
			default:
				return false, s
			}
		},
		Equal: func(a, b any) bool {
			return a.(int64) == b.(int64)
		},
		DescribeOperation: func(input, output any) string {
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			return fmt.Sprintf("%s(%v) -> %v", inp.Name, inp.Args, out.Result)
		},
		DescribeState: func(state any) string {
			return strconv.FormatInt(state.(int64), 10)
		},
	}
}

// counterDelta interprets the Inc/Dec args as a signed integer. A
// nil args slot (or a missing one) defaults to 1.
func counterDelta(args any) int64 {
	switch v := args.(type) {
	case nil:
		return 1
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	default:
		return 1
	}
}

// matchesInt64 reports whether out is an int64-convertible value
// equal to want. Accepts int / int32 / int64 to absorb the typical
// counter-return types.
func matchesInt64(out any, want int64) bool {
	switch v := out.(type) {
	case int64:
		return v == want
	case int:
		return int64(v) == want
	case int32:
		return int64(v) == want
	default:
		return false
	}
}

// singleHistoryPartition assigns every operation to a single
// shared partition. Used by Counter, AppendLog, and Pool, where
// the model state is one global value rather than per-key.
func singleHistoryPartition(history []porcupine.Operation) [][]porcupine.Operation {
	if len(history) == 0 {
		return nil
	}
	return [][]porcupine.Operation{history}
}
