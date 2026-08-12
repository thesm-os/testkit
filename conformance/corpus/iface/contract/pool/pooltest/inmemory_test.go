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
// Put is classified writer, so the harness seeds through it — which is what
// makes Get's smoke check meet a pool with something in it. An empty pool would
// exercise only the exhausted path, and the contract's ordinary case would go
// unrun.
func TestContractContract(t *testing.T) {
	t.Parallel()

	pooltest.AssertContractContract(t,
		pooltest.ContractModel(),
		pooltest.ContractSubject("in-memory", func() pool.Contract {
			return pooltest.NewInMemory()
		}),
		pooltest.ContractOnGet("hands out what it holds and no more", handsOutWhatItHolds),
	)
}

// handsOutWhatItHolds is the pool's bound, stated through the interface.
//
// It ends on a refusal rather than on a successful Get, and that is what makes
// it a check: the seed put exactly one value in, so a pool that manufactures on
// demand — or one whose methods return nil and nothing else — answers the
// second Get too. [TestEveryCheckRejectsAnUnboundedPool] drives exactly that.
func handsOutWhatItHolds(tb testing.TB, subject pool.Contract) {
	tb.Helper()
	got, err := subject.Get(tb.Context())
	testkit.NoError(tb, err, "the seeded value is available")

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

// The check rejects a pool that does not bound anything.
//
// The message is asserted as well as the rejection: a stand-in failing for some
// unrelated reason would satisfy a boolean guard while the check's own
// assertion never ran.
func TestEveryCheckRejectsAnUnboundedPool(t *testing.T) {
	t.Parallel()

	got := testkit.Rejects(t, "a pool with no bound", func(tb testing.TB) {
		tb.Helper()
		handsOutWhatItHolds(tb, unboundedPool{})
	})
	testkit.Assert(t, got).Contains("the pool it came from is then empty",
		"rejected for the reason the check is about")
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	pooltest.AssertContractContract(t,
		pooltest.ContractSubject("in-memory", func() pool.Contract {
			return pooltest.NewInMemory()
		}),
		pooltest.ContractWithout("Get/smoke"),
		pooltest.ContractWithoutDouble(),
	)
}
