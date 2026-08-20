// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package orderaftertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/orderafter"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/orderafter/orderaftertest"
)

// The one classification here whose partner the generator can name.
//
// `//testkit:mixin orderafter fn=Prepare` is resolved to a qualified name by the
// shape resolver and cut back to its local form for the call site — which is why
// this check exists and `sideeffect`, `partition`, `hooks` and `sample` do not:
// those describe a relationship between two methods and declare no parameter
// naming the second.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	orderaftertest.RunMixed(t,
		orderaftertest.MixedHarness[*orderaftertest.InMemory]{Name: "in-memory", New: orderaftertest.NewInMemory},
		orderaftertest.MixedChecks{
			{
				Method: "Commit",
				Name:   "accepted-after-the-prerequisite",
				Claim:  "Commit succeeds once the prerequisite has run",
				Run: func(tb testing.TB, s orderafter.Mixed, fx orderaftertest.MixedFixture) {
					tb.Helper()
					// Both halves, because the ordering claim needs both: that
					// a commit is refused before Prepare, and that it is
					// accepted after. Nothing says the constraint is the only
					// reason a commit could fail, so the second half is not
					// implied by the first.
					testkit.Error(tb, s.Commit(tb.Context()),
						"committing before the prerequisite is refused")

					testkit.NoError(tb, s.Prepare(tb.Context()), "preparing succeeds")
					testkit.NoError(tb, s.Commit(tb.Context()), "and then so does committing")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	orderaftertest.RunMixed(t,
		orderaftertest.MixedHarness[*orderaftertest.InMemory]{Name: "in-memory", New: orderaftertest.NewInMemory},
		orderaftertest.MixedSuite.Without(orderaftertest.MixedSuite.Checks.Commit.Smoke()),
	)
}
