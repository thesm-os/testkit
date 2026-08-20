// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// pool is the model tier's under ADR-0028: `AUTO-POOL-BALANCED` and
// `AUTO-POOL-LEAK-FREE` state it, and both are claims about a sequence rather
// than about a call.
//
// The row below is the bound, stated through the interface: it puts one value
// in and asks for two.
package pooltest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pool"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pool/pooltest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	pooltest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	pooltest.RunContract(t,
		inMemory("in-memory"),
		pooltest.ContractSuite.Without(pooltest.ContractSuite.Checks.Get.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	pooltest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

// inMemory is the empty pool a run starts from; values arrive through Put.
//
// The constructor is variadic, so the harness closes over it rather than naming
// it: a seeded pool would answer the second Get for a reason the row is not
// about.
func inMemory(name string) pooltest.ContractHarness[*pooltest.InMemory] {
	return pooltest.ContractHarness[*pooltest.InMemory]{
		Name: name,
		New:  func() *pooltest.InMemory { return pooltest.NewInMemory() },
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = pooltest.ContractChecks{
	{
		Method: "Get", Name: "hands-out-what-it-holds",
		Claim: "Get hands out what it holds and no more",
		Run:   handsOutWhatItHolds,
		ProvenBy: pooltest.BrokenContract(
			"a pool with no bound", newUnboundedPool,
		),
		ProvenReason: "the pool it came from is then empty",
	},
}

// --- Bodies -------------------------------------------------------------------

// handsOutWhatItHolds is the pool's bound, stated through the interface.
//
// It ends on a refusal rather than on a successful Get, and that is what makes
// it a check: exactly one value goes in, so a pool that manufactures on demand —
// or one whose methods return nil and nothing else — answers the second Get too.
func handsOutWhatItHolds(
	tb testing.TB, s pool.Contract, fx pooltest.ContractFixture,
) {
	tb.Helper()
	testkit.NoError(tb, s.Put(tb.Context(), fx.Value()), "one value goes in")

	got, err := s.Get(tb.Context())
	testkit.NoError(tb, err, "and is available")

	_, err = s.Get(tb.Context())
	testkit.Error(tb, err, "and the pool it came from is then empty")

	testkit.NoError(tb, s.Put(tb.Context(), got), "returning it succeeds")
	_, err = s.Get(tb.Context())
	testkit.NoError(tb, err, "and the pool can hand it out again")
}

// --- Planted defects ----------------------------------------------------------

// unboundedPool hands out a fresh value however often it is asked, which is a
// constructor rather than a pool.
//
// The implementation every check on Get exists to reject: it satisfies any
// assertion that a Get succeeded, and provides no limit at all.
type unboundedPool struct{}

func newUnboundedPool() unboundedPool { return unboundedPool{} }

func (unboundedPool) Get(context.Context) (pool.Value, error) { return pool.Value{}, nil }
func (unboundedPool) Put(context.Context, pool.Value) error   { return nil }

// Stats lies the way the rest of the stand-in does: permanently balanced
// numbers from a pool that bounds nothing.
func (unboundedPool) Stats(context.Context) (pool.Stats, error) { return pool.Stats{}, nil }
