// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"fmt"
	"strconv"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit/model"
)

// poolState tracks the running outstanding count for a pool. State
// is shared across the whole history.
type poolState struct {
	outstanding int
	gets        int
	puts        int
}

// Pool builds a porcupine.Model for resource-pool interfaces. State
// is the running outstanding count (gets - puts). Step functions:
//
//   - Get: succeeds (no error); state.outstanding++.
//   - Put: succeeds iff outstanding > 0 (no double-Put); state
//     .outstanding--.
//
// At quiescence the model expects outstanding == 0 for the
// leak-free contract; this is verified by the runner outside the
// linearizability check.
func Pool() porcupine.Model {
	return porcupine.Model{
		Partition: singleHistoryPartition,
		Init:      func() any { return poolState{} },
		Step: func(state, input, output any) (bool, any) {
			s := state.(poolState)
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			r, ok := out.Result.(WriterResult)
			if !ok {
				return false, s
			}

			switch inp.Name {
			case OpGet:
				if r.Err != nil {
					return false, s
				}
				return true, poolState{outstanding: s.outstanding + 1, gets: s.gets + 1, puts: s.puts}
			case OpPut:
				if s.outstanding == 0 {
					return r.Err != nil, s // double-Put must error
				}
				if r.Err != nil {
					return false, s
				}
				return true, poolState{outstanding: s.outstanding - 1, gets: s.gets, puts: s.puts + 1}
			default:
				return false, s
			}
		},
		Equal: func(a, b any) bool {
			return a.(poolState) == b.(poolState)
		},
		DescribeOperation: func(input, output any) string {
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			return fmt.Sprintf("%s -> %v", inp.Name, out.Result)
		},
		DescribeState: func(state any) string {
			s := state.(poolState)
			return "outstanding=" + strconv.Itoa(s.outstanding)
		},
	}
}
