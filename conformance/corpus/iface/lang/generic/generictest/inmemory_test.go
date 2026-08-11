// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generictest_test

import (
	"context"
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
func TestStoreContract(t *testing.T) {
	t.Parallel()

	generictest.AssertStoreContract[string, int](t,
		generictest.StoreSubject[string, int]("in-memory", func() generic.Store[string, int] {
			return generictest.NewInMemory[string, int]()
		}),
	)
}

// The fixture is empty by construction: a type parameter admits no literal, so
// K and V stay at their zero values and the miss check that reads them is not
// generated. What a consumer supplies is exactly what derivation could not.
func TestStoreContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	fixture := generictest.DefaultStoreFixture[string, int]()
	fixture.Key = "seeded-key"
	fixture.KeyOther = "absent-key"
	fixture.Value = 7

	generictest.AssertStoreContract[string, int](t,
		generictest.StoreSubject[string, int]("supplied", func() generic.Store[string, int] {
			return generictest.NewInMemory[string, int]()
		}),
		generictest.StoreWithFixture[string, int](fixture),
		generictest.StoreSeed[string, int](func(ctx context.Context, subject generic.Store[string, int]) error {
			return subject.Put(ctx, fixture.Key, fixture.Value)
		}),
		generictest.StoreOnGet[string, int]("returns what was seeded", func(
			tb testing.TB, subject generic.Store[string, int], key string,
		) {
			tb.Helper()
			got, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "a seeded key is found")
			testkit.Equal(tb, got, fixture.Value, "and carries what was written")
		}),
		generictest.StoreWithout[string, int]("Put/smoke"),
		generictest.StoreWithoutDouble[string, int](),
	)
}
