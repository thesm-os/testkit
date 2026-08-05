// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit/engine/model"
)

// NondeterministicModel wraps another porcupine.Model so a marked
// subset of operations may succeed-or-not under fault injection
// without rejecting the history. The trace recorder wraps an
// operation's Args in a NondeterministicOutcome when a fault was
// active during the call; this wrapper accepts either the success
// transition (delegated to the inner model with the unwrapped args)
// or a no-op leaving state unchanged.
//
// Without a fault marker on the args the wrapper delegates directly
// to the inner model.
func NondeterministicModel(inner porcupine.Model) porcupine.Model {
	return porcupine.Model{
		Partition: inner.Partition,
		Init:      inner.Init,
		Step: func(state, input, output any) (bool, any) {
			inp, ok := input.(model.OpInput)
			if !ok {
				return inner.Step(state, input, output)
			}
			nd, marked := inp.Args.(NondeterministicOutcome)
			if !marked {
				return inner.Step(state, input, output)
			}
			unwrapped := model.OpInput{Name: inp.Name, Args: nd.Inner, PartitionKey: inp.PartitionKey}
			if okStep, next := inner.Step(state, unwrapped, output); okStep {
				return true, next
			}
			return true, state
		},
		Equal:             inner.Equal,
		DescribeOperation: inner.DescribeOperation,
		DescribeState:     inner.DescribeState,
	}
}

// NondeterministicOutcome marks an input as fault-affected. The
// Inner field carries the original input the consumer would have
// passed without the fault marker.
type NondeterministicOutcome struct {
	Inner any
}
