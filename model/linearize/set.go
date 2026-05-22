// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"fmt"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit/model"
)

// setMembership is the per-partition state for a single element.
type setMembership struct {
	present bool
}

// Set builds a porcupine.Model for set-shaped interfaces partitioned
// by element. Each element is independently present or absent.
// Step functions:
//
//   - Add:      output is the prior membership bool (true if already
//     present), state becomes present.
//   - Remove:   output is the prior membership bool (true if was
//     present), state becomes absent.
//   - Contains: output equals the current membership.
//
// The partition key comes from model.OpInput.PartitionKey. Operation
// names are fixed; the generator emits aliases when SUT method names
// differ.
func Set() porcupine.Model {
	return porcupine.Model{
		Partition: partitionByKey,
		Init:      func() any { return setMembership{} },
		Step: func(state, input, output any) (bool, any) {
			s := state.(setMembership)
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			gotPrior, ok := out.Result.(bool)
			if !ok && inp.Name != "Contains" {
				return false, s
			}

			switch inp.Name {
			case "Add":
				if gotPrior != s.present {
					return false, s
				}
				return true, setMembership{present: true}
			case "Remove":
				if gotPrior != s.present {
					return false, s
				}
				return true, setMembership{present: false}
			case "Contains":
				gotNow, ok := out.Result.(bool)
				if !ok {
					return false, s
				}
				return gotNow == s.present, s
			default:
				return false, s
			}
		},
		Equal: func(a, b any) bool {
			return a.(setMembership) == b.(setMembership)
		},
		DescribeOperation: func(input, output any) string {
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			return fmt.Sprintf("%s -> %v", inp.Name, out.Result)
		},
		DescribeState: func(state any) string {
			if state.(setMembership).present {
				return "<in>"
			}
			return "<out>"
		},
	}
}
