// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package pooltest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pool"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pool/pooltest"
)

// pool is the model tier's under ADR-0018: `AUTO-POOL-BALANCED` and
// `AUTO-POOL-LEAK-FREE` state it, and both are claims about a sequence rather
// than about a call.
//
// The row below is the bound, stated through the interface: it puts one value
// in and asks for two.
func TestContractContract(t *testing.T) {
	t.Parallel()

	pooltest.RunContract(t,
		pooltest.ContractHarness[*pooltest.InMemory]{
			Name: "in-memory",
			New:  func() *pooltest.InMemory { return pooltest.NewInMemory() },
		},
		pooltest.ContractChecks{
			{
				Method: "Get",
				Name:   "hands-out-what-it-holds",
				Claim:  "Get hands out what it holds and no more",
				Run: func(tb testing.TB, s pool.Contract, fx pooltest.ContractFixture) {
					tb.Helper()
					handsOutWhatItHolds(tb, s, fx.Value())
				},
			},
		},
	)
}

// handsOutWhatItHolds is the pool's bound, stated through the interface.
//
// It ends on a refusal rather than on a successful Get, and that is what makes
// it a check: exactly one value goes in, so a pool that manufactures on demand —
// or one whose methods return nil and nothing else — answers the second Get too.
// [TestEveryCheckRejectsAnUnboundedPool] drives exactly that.
func handsOutWhatItHolds(tb testing.TB, subject pool.Contract, seed pool.Value) {
	tb.Helper()
	testkit.NoError(tb, subject.Put(tb.Context(), seed), "one value goes in")

	got, err := subject.Get(tb.Context())
	testkit.NoError(tb, err, "and is available")

	_, err = subject.Get(tb.Context())
	testkit.Error(tb, err, "and the pool it came from is then empty")

	testkit.NoError(tb, subject.Put(tb.Context(), got), "returning it succeeds")
	_, err = subject.Get(tb.Context())
	testkit.NoError(tb, err, "and the pool can hand it out again")
}

// unboundedPool hands out a fresh value however often it is asked, which is a
// constructor rather than a pool.
//
// The implementation every check on Get exists to reject: it satisfies any
// assertion that a Get succeeded, and provides no limit at all.
type unboundedPool struct{}

func (unboundedPool) Get(context.Context) (pool.Value, error) { return pool.Value{}, nil }
func (unboundedPool) Put(context.Context, pool.Value) error   { return nil }

// Stats lies the way the rest of the stand-in does: permanently
// balanced numbers from a pool that bounds nothing.
func (unboundedPool) Stats(context.Context) (pool.Stats, error) { return pool.Stats{}, nil }

// The check rejects a pool that does not bound anything.
//
// The message is asserted as well as the rejection: a stand-in failing for some
// unrelated reason would satisfy a boolean guard while the check's own
// assertion never ran.
func TestEveryCheckRejectsAnUnboundedPool(t *testing.T) {
	t.Parallel()

	got := testkit.Rejects(t, "a pool with no bound", func(tb testing.TB) {
		tb.Helper()
		handsOutWhatItHolds(tb, unboundedPool{}, pooltest.DefaultContractFixture().Value())
	})
	testkit.Assert(t, got).Contains("the pool it came from is then empty",
		"rejected for the reason the check is about")
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	pooltest.RunContract(t,
		pooltest.ContractHarness[*pooltest.InMemory]{
			Name: "in-memory",
			New:  func() *pooltest.InMemory { return pooltest.NewInMemory() },
		},
		pooltest.ContractSuite.Without(pooltest.ContractSuite.Checks.Get.Smoke()),
	)
}
