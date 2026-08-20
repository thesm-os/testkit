// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generictest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/generic"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/generic/generictest"
)

// A generic interface's harness is generic, and the consumer instantiates it.
//
// A Go test function cannot take type parameters, so the harness cannot be one
// — but every declaration it emits can, and naming the types here is the same
// thing a consumer does when they construct the implementation. Nothing is
// derived at witnesses: the caller already knows which instantiation they run.
//
// Which is also why nothing is derived at all: a type parameter admits no
// literal, so every family the rules reached was refused and the header lists
// them. The values come from the row, which is the one place they can.
func TestStoreContract(t *testing.T) {
	t.Parallel()

	generictest.RunStore[string, int](t,
		generictest.StoreHarness[string, int, *generictest.InMemory[string, int]]{
			Name: "in-memory",
			New:  generictest.NewInMemory[string, int],
		},
		generictest.StoreChecks[string, int]{
			{
				Method: "Get",
				Name:   "reads-back-what-put-wrote",
				Claim:  "Get returns what Put wrote",
				Run: func(tb testing.TB, s generic.Store[string, int], fx generictest.StoreFixture[string, int]) {
					tb.Helper()
					// The values are the row's: the fixture's K and V are the
					// type parameters' zeros, because no literal can be written
					// for a type nobody has instantiated yet.
					testkit.NoError(tb, s.Put(tb.Context(), "seeded-key", 7), "the key is written")

					got, err := s.Get(tb.Context(), "seeded-key")
					testkit.NoError(tb, err, "a written key is found")
					testkit.Equal(tb, got, 7, "and carries what was written")
				},
			},
			{
				Method: "Get",
				Name:   "miss-is-reported",
				Claim:  "Get reports a key nothing wrote",
				Run: func(tb testing.TB, s generic.Store[string, int], fx generictest.StoreFixture[string, int]) {
					tb.Helper()
					got, err := s.Get(tb.Context(), "absent-key")
					testkit.Error(tb, err, "an unwritten key is a miss")
					testkit.Equal(tb, got, 0, "and the value beside it is the zero")
				},
			},
		},
	)
}
