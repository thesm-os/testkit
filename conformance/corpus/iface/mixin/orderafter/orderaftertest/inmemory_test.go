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
// naming the second (thesm-os/eidos#16).
func TestMixedContract(t *testing.T) {
	t.Parallel()

	orderaftertest.AssertMixedContract(t,
		orderaftertest.MixedSubject("in-memory", func() orderafter.Mixed {
			return orderaftertest.NewInMemory()
		}),
		orderaftertest.MixedOnCommit("succeeds once the prerequisite has run", func(
			tb testing.TB, subject orderafter.Mixed,
		) {
			tb.Helper()
			// The other half. The generated check states that Commit is refused
			// before Prepare; that it is *accepted* after is the half a
			// classification cannot imply, since nothing says the constraint is
			// the only reason a commit could fail.
			testkit.NoError(tb, subject.Prepare(tb.Context()), "preparing succeeds")
			testkit.NoError(tb, subject.Commit(tb.Context()), "and then so does committing")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	orderaftertest.AssertMixedContract(t,
		orderaftertest.MixedSubject("in-memory", func() orderafter.Mixed {
			return orderaftertest.NewInMemory()
		}),
		orderaftertest.MixedWithout("Commit/smoke"),
		orderaftertest.MixedWithoutDouble(),
	)
}
