// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package namedreturnstest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/namedreturns"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/namedreturns/namedreturnstest"
)

// Whether the source named its results changes the declaration and nothing
// else, so all three spellings are held to one contract.
//
// The author's own rule rides along in the same run: that the three agree.
// No classification says so — it is a fact about this interface that a reader
// of the source would assume and nothing would check.
func TestServiceContract(t *testing.T) {
	t.Parallel()

	fx := namedreturnstest.DefaultServiceFixture()

	namedreturnstest.RunService(t,
		namedreturnstest.ServiceHarness[*namedreturnstest.InMemory]{
			Name: "in-memory",
			// Nothing on this interface writes, so the seed is the
			// constructor's. The key comes off the fixture rather than being
			// written out, so it and the row cannot disagree.
			New: func() *namedreturnstest.InMemory {
				s := namedreturnstest.NewInMemory()
				s.Put(fx.ID(), "stored")
				return s
			},
		},
		namedreturnstest.ServiceChecks{
			{
				Method: "Named",
				Name:   "three-spellings-agree",
				Claim:  "Named agrees with the other two spellings",
				Run: func(tb testing.TB, s namedreturns.Service, fx namedreturnstest.ServiceFixture) {
					tb.Helper()
					named, err := s.Named(tb.Context(), fx.ID())
					testkit.NoError(tb, err, "a seeded identifier is found")

					unnamed, _ := s.Unnamed(tb.Context(), fx.ID())
					partial, _ := s.PartiallyNamed(tb.Context(), fx.ID())
					testkit.Equal(tb, unnamed, named, "the anonymous form answers alike")
					testkit.Equal(tb, partial, named, "and so does the partially named one")
				},
			},
		},
	)
}

// Dropping a check is how a consumer keeps a suite they would otherwise
// abandon. Run here against the same subject, because what is under test is
// the suppression rather than the implementation.
func TestServiceContractSuppression(t *testing.T) {
	t.Parallel()

	namedreturnstest.RunService(
		t,
		namedreturnstest.ServiceHarness[*namedreturnstest.InMemory]{
			Name: "in-memory",
			New:  namedreturnstest.NewInMemory,
		},
		namedreturnstest.ServiceSuite.Without(namedreturnstest.ServiceSuite.Checks.Named.Smoke()),
	)
}
