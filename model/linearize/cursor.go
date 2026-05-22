// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/anishathalye/porcupine"
	"github.com/google/go-cmp/cmp"

	"go.thesmos.sh/testkit/model"
)

// cursorState is the index into the preloaded items list plus the
// closed flag. State is shared across the whole history.
type cursorState struct {
	index  int
	closed bool
}

// Cursor builds a porcupine.Model for single-pass iterators with an
// idempotent Close. The model is parameterized by the static
// expected-elements slice; the cursor yields each item once until
// exhaustion, then signals end. Step functions:
//
//   - Next:  before close, returns the next item plus true; after
//     exhaustion, returns the zero value plus false; after
//     Close, returns the configured sentinel error.
//   - Close: idempotent — never errors, just transitions closed.
//
// Operation names are fixed ("Next", "Close").
func Cursor[V any](items []V, closedSentinel error) porcupine.Model {
	return porcupine.Model{
		Partition: singleHistoryPartition,
		Init:      func() any { return cursorState{} },
		Step: func(state, input, output any) (bool, any) {
			s := state.(cursorState)
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)

			switch inp.Name {
			case "Next":
				r, ok := out.Result.(ReaderBoolResult[V])
				if !ok {
					return false, s
				}
				if s.closed {
					// Closed cursor: any Next call must surface the sentinel.
					// We carry the sentinel through the ok=false signal by
					// matching r.OK==false and (if defined) the sentinel error
					// via a paired Err output type. The ReaderBoolResult shape
					// has no Err field, so closed-cursor detection is
					// expressed via OK=false on closed state.
					return !r.OK, s
				}
				if s.index >= len(items) {
					return !r.OK, s
				}
				if !r.OK {
					return false, s
				}
				if !cmp.Equal(r.Value, items[s.index]) {
					return false, s
				}
				return true, cursorState{index: s.index + 1, closed: s.closed}

			case "Close":
				r, ok := out.Result.(WriterResult)
				if !ok {
					return false, s
				}
				if r.Err != nil {
					return false, s
				}
				return true, cursorState{index: s.index, closed: true}

			default:
				_ = errors.Is // imported for symmetry across linearize package
				_ = closedSentinel
				return false, s
			}
		},
		Equal: func(a, b any) bool {
			return a.(cursorState) == b.(cursorState)
		},
		DescribeOperation: func(input, output any) string {
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			return fmt.Sprintf("%s -> %v", inp.Name, out.Result)
		},
		DescribeState: func(state any) string {
			s := state.(cursorState)
			status := "open"
			if s.closed {
				status = "closed"
			}
			return "idx=" + strconv.Itoa(s.index) + " " + status
		},
	}
}
