// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package hookstest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/hooks"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/hooks/hookstest"
)

// The only generated check that constructs the thing it passes.
//
// A registration takes a callback, so the check has to build one — and the
// callback's own signature is what a func literal declares. It comes off the
// partner's func-typed parameter, spelled as types without names, which is all
// a literal needs and avoids inventing identifiers the body ignores.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	hookstest.AssertMixedContract(t,
		hookstest.MixedModel(),
		hookstest.MixedSubject("in-memory", func() hooks.Mixed {
			return hookstest.NewInMemory()
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	hookstest.AssertMixedContract(t,
		hookstest.MixedSubject("in-memory", func() hooks.Mixed {
			return hookstest.NewInMemory()
		}),
		hookstest.MixedWithout("Fire/smoke"),
		hookstest.MixedWithoutDouble(),
	)
}
