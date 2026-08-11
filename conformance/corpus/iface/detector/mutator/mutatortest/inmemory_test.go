// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutatortest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/mutator"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/mutator/mutatortest"
)

// A write returning nothing earns two checks: the smoke call and nil-context
// tolerance. Cancellation and deadline are claims about what a method reports,
// and this one reports nothing.
//
// It also cannot seed. The suite derives its seed from the writer detectors, and
// mutator is deliberately excluded from that set for exactly this reason — a
// seed through a void method cannot say whether it worked, which would leave
// every later check running against an empty subject and passing.
func TestMutatorContract(t *testing.T) {
	t.Parallel()

	mutatortest.AssertMutatorContract(t,
		mutatortest.MutatorSubject("in-memory", func() mutator.Mutator {
			return mutatortest.NewInMemory()
		}),
	)
}

// The effect is out of band, so observing it needs a method the interface does
// not declare — which is why no generated check can make this claim.
func TestTouchRecordsAndDeclinesADoneContext(t *testing.T) {
	t.Parallel()

	s := mutatortest.NewInMemory()
	s.Touch(t.Context(), "k")
	testkit.Equal(t, s.Touches("k"), 1, "a touch is recorded")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	s.Touch(ctx, "k")
	testkit.Equal(t, s.Touches("k"), 1,
		"and a cancelled one does no work, which is the only refusal open to it")
}

// Declining the double is separate from dropping a check.
func TestMutatorContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	mutatortest.AssertMutatorContract(t,
		mutatortest.MutatorSubject("in-memory", func() mutator.Mutator {
			return mutatortest.NewInMemory()
		}),
		mutatortest.MutatorWithout("Touch/smoke"),
		mutatortest.MutatorWithoutDouble(),
	)
}
