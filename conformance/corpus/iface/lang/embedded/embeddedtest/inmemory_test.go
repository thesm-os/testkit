// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package embeddedtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/embedded"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/embedded/embeddedtest"
)

// An interface's declarations are not its method set. Composed declares Get and
// inherits Ping and Close, so a harness reading declarations alone would hold an
// implementation to a third of its contract and report success.
func TestComposedContract(t *testing.T) {
	t.Parallel()

	fx := embeddedtest.DefaultComposedFixture()

	embeddedtest.RunComposed(t,
		embeddedtest.ComposedHarness[*embeddedtest.InMemory]{
			Name: "in-memory",
			// Composed declares no writer, so the reader's hit path is
			// unreachable without a seeded constructor.
			New: func() *embeddedtest.InMemory {
				s := embeddedtest.NewInMemory()
				s.Put(fx.Key(), "seeded")
				return s
			},
		},
		embeddedtest.ComposedChecks{
			{
				Method: "Get",
				Name:   "returns-what-was-seeded",
				Claim:  "Get returns what was seeded",
				Run: func(tb testing.TB, s embedded.Composed, fx embeddedtest.ComposedFixture) {
					tb.Helper()
					got, err := s.Get(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a seeded identifier is found")
					testkit.Equal(tb, got, "seeded", "and carries what was written")
				},
			},
		},
	)
}

// The embedded interfaces are contracts in their own right, and one
// implementation answers to all three.
//
// No seed. Base exposes no state a reader observes, so its constructor is the
// bare one.
func TestBaseContract(t *testing.T) {
	t.Parallel()

	embeddedtest.RunBase(t,
		embeddedtest.BaseHarness[*embeddedtest.InMemory]{Name: "in-memory", New: embeddedtest.NewInMemory},
		embeddedtest.BaseChecks{
			{
				Method: "Ping",
				Name:   "succeeds-on-a-fresh-subject",
				Claim:  "Ping succeeds on a fresh subject",
				Run: func(tb testing.TB, s embedded.Base, fx embeddedtest.BaseFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Ping(tb.Context()), "an open subject answers a ping")
				},
			},
		},
	)
}

// Dropping a check is per contract rather than per package: three interfaces
// share this file and each has its own index.
func TestBaseContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	embeddedtest.RunBase(t,
		embeddedtest.BaseHarness[*embeddedtest.InMemory]{Name: "in-memory", New: embeddedtest.NewInMemory},
		embeddedtest.BaseSuite.Without(embeddedtest.BaseSuite.Checks.Ping.Smoke()),
	)
}

func TestCloserContract(t *testing.T) {
	t.Parallel()

	embeddedtest.RunCloser(t,
		embeddedtest.CloserHarness[*embeddedtest.InMemory]{Name: "in-memory", New: embeddedtest.NewInMemory},
		embeddedtest.CloserChecks{
			{
				Method: "Close",
				Name:   "second-close-succeeds",
				Claim:  "Close is idempotent",
				Run: func(tb testing.TB, s embedded.Closer, fx embeddedtest.CloserFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Close(tb.Context()), "the first close succeeds")
					testkit.NoError(tb, s.Close(tb.Context()), "and so does the second")
				},
			},
		},
	)
}

// The same, for Closer.
func TestCloserContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	embeddedtest.RunCloser(t,
		embeddedtest.CloserHarness[*embeddedtest.InMemory]{Name: "in-memory", New: embeddedtest.NewInMemory},
		embeddedtest.CloserSuite.Without(embeddedtest.CloserSuite.Checks.Close.Smoke()),
	)
}
