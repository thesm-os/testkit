// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package namedreturnstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/namedreturns"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/namedreturns/namedreturnstest"
)

// Whether the source named its results changes the declaration and nothing
// else, so all three spellings are held to one contract.
//
// The author's own rule rides along in the same call: that the three agree.
// No classification says so — it is a fact about this interface that a reader
// of the source would assume and nothing would check.
func TestServiceContract(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced: nothing here
	// passes ServiceWithFixture, so the derivation stands.
	fixture := namedreturnstest.DefaultServiceFixture()

	namedreturnstest.AssertServiceContract(t,
		namedreturnstest.ServiceSubject("in-memory", func() namedreturns.Service {
			return namedreturnstest.NewInMemory()
		}),
		namedreturnstest.ServiceSeed(func(_ context.Context, subject namedreturns.Service) error {
			// The key comes off the fixture rather than being written out, so
			// the seed and the checks cannot disagree about what was stored.
			subject.(*namedreturnstest.InMemory).Put(fixture.ID, "stored")
			return nil
		}),
		namedreturnstest.ServiceOnNamed("agrees with the other two spellings", func(
			tb testing.TB, subject namedreturns.Service, id string,
		) {
			tb.Helper()
			named, err := subject.Named(tb.Context(), id)
			testkit.NoError(tb, err, "a seeded identifier is found")

			unnamed, _ := subject.Unnamed(tb.Context(), id)
			partial, _ := subject.PartiallyNamed(tb.Context(), id)
			testkit.Equal(tb, unnamed, named, "the anonymous form answers alike")
			testkit.Equal(tb, partial, named, "and so does the partially named one")
		}),
	)
}

// Dropping a check is how a consumer keeps a suite they would otherwise
// abandon, and declining the double is how they keep one when they do not use
// it. Both are run here against the same subject, because what is under test is
// the suppression rather than the implementation.
func TestServiceContractSuppression(t *testing.T) {
	t.Parallel()

	namedreturnstest.AssertServiceContract(t,
		namedreturnstest.ServiceSubject("in-memory", func() namedreturns.Service {
			return namedreturnstest.NewInMemory()
		}),
		namedreturnstest.ServiceWithout("Named/smoke"),
		namedreturnstest.ServiceWithoutDouble(),
	)
}
