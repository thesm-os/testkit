// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sideeffecttest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sideeffect"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sideeffect/sideeffecttest"
)

// The first generated check that calls a second method, and the first that
// could not be generated at all until eidos let the mixin name one.
//
// `//testkit:mixin sideeffect observe=Observed` is the whole of the wiring. The
// resolver rewrites Observed into a qualified name, the projection cuts it back
// to the local form a call site can spell, and the check observes either side of
// the call. Without the parameter the mixin said only *that* there was an effect
// — which is not a claim a test can make.
//
// So there is nothing hand-written here about the effect. The subject, the
// contract and one option is the whole file.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	sideeffecttest.AssertMixedContract(t,
		sideeffecttest.MixedModel(),
		sideeffecttest.MixedSubject("in-memory", func() sideeffect.Mixed {
			return sideeffecttest.NewInMemory()
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	sideeffecttest.AssertMixedContract(t,
		sideeffecttest.MixedSubject("in-memory", func() sideeffect.Mixed {
			return sideeffecttest.NewInMemory()
		}),
		sideeffecttest.MixedWithout(
			"Touch/smoke",
		),
		sideeffecttest.MixedWithoutDouble(),
	)
}
