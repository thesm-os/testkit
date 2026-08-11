// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package hookstest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
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
		hookstest.MixedSubject("in-memory", func() hooks.Mixed {
			return hookstest.NewInMemory()
		}),
	)
}

// A handler firing during Fire may register another, which is ordinary rather
// than exotic — and deadlocks a subject that holds its lock across the calls.
// No generated check reaches this: it needs a callback that reaches back.
func TestFireToleratesReentrantRegistration(t *testing.T) {
	t.Parallel()

	s := hookstest.NewInMemory()
	s.OnEvent(func(string) {
		s.OnEvent(func(string) {})
	})

	testkit.NoError(t, s.Fire(t.Context(), "e"), "firing a re-entrant handler completes")
	testkit.Equal(t, s.Registered(), 2, "and the handler it registered is attached")
}

// A nil handler is dropped rather than stored, or the next Fire panics on
// something a caller passed by accident.
func TestOnEventIgnoresNil(t *testing.T) {
	t.Parallel()

	s := hookstest.NewInMemory()
	s.OnEvent(nil)
	testkit.Equal(t, s.Registered(), 0, "nothing was registered")
	testkit.NoError(t, s.Fire(t.Context(), "e"), "and firing does not reach it")
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
