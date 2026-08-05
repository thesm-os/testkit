// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"errors"
	"fmt"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit/engine/model"
)

// leaseState is the per-partition state for a single lease key.
type leaseState struct {
	held bool
}

// LeaseTable builds a porcupine.Model for lease-tracking interfaces
// partitioned by key. Each key holds at most one lease at a time:
//
//   - Acquire on an unheld key succeeds (no error) and transitions
//     the state to held.
//   - Acquire on a held key fails with the configured held error and
//     does not change state.
//   - Release on a held key succeeds (no error) and transitions to
//     unheld.
//   - Release on an unheld key fails with the configured free error
//     and does not change state.
//
// The partition key comes from model.OpInput.PartitionKey. heldErr
// and freeErr are matched via errors.Is; either may be nil to mean
// "any non-nil error counts."
func LeaseTable(heldErr, freeErr error) porcupine.Model {
	return porcupine.Model{
		Partition: partitionByKey,
		Init:      func() any { return leaseState{} },
		Step: func(state, input, output any) (bool, any) {
			s := state.(leaseState)
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			r, ok := out.Result.(WriterResult)
			if !ok {
				return false, s
			}

			switch inp.Name {
			case "Acquire":
				if s.held {
					return matchErr(r.Err, heldErr), s
				}
				if r.Err != nil {
					return false, s
				}
				return true, leaseState{held: true}

			case "Release":
				if !s.held {
					return matchErr(r.Err, freeErr), s
				}
				if r.Err != nil {
					return false, s
				}
				return true, leaseState{held: false}

			default:
				return false, s
			}
		},
		Equal: func(a, b any) bool {
			return a.(leaseState) == b.(leaseState)
		},
		DescribeOperation: func(input, output any) string {
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			return fmt.Sprintf("%s -> %v", inp.Name, out.Result)
		},
		DescribeState: func(state any) string {
			if state.(leaseState).held {
				return "<held>"
			}
			return "<free>"
		},
	}
}

// matchErr reports whether got satisfies the want sentinel. When
// want is nil, any non-nil error is acceptable; otherwise got must
// wrap want via errors.Is.
func matchErr(got, want error) bool {
	if want == nil {
		return got != nil
	}
	return errors.Is(got, want)
}
