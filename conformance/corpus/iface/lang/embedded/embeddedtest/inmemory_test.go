// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package embeddedtest_test

import (
	"context"
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

	// The values the run itself uses, read rather than replaced: nothing here
	// passes ComposedWithFixture, so the derivation stands.
	fixture := embeddedtest.DefaultComposedFixture()

	embeddedtest.AssertComposedContract(t,
		embeddedtest.ComposedSubject("in-memory", func() embedded.Composed {
			return embeddedtest.NewInMemory()
		}),
		embeddedtest.ComposedSeed(func(_ context.Context, subject embedded.Composed) error {
			// A seed may reach for the concrete subject: it runs before the
			// double wraps it and sees what the factory made. A check may not.
			subject.(*embeddedtest.InMemory).Put(fixture.Key, "seeded")
			return nil
		}),
		embeddedtest.ComposedOnGet("returns what was seeded", func(
			tb testing.TB, subject embedded.Composed, key string,
		) {
			tb.Helper()
			got, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "a seeded identifier is found")
			testkit.Equal(tb, got, "seeded", "and carries what was written")
		}),
	)
}

// The embedded interfaces are contracts in their own right, and one
// implementation answers to all three.
//
// No fixture and no seed. Base exposes no state a reader observes, and passing
// a seed that returns nil would say the same thing the absence already says.
func TestBaseContract(t *testing.T) {
	t.Parallel()

	embeddedtest.AssertBaseContract(t,
		embeddedtest.BaseSubject("in-memory", func() embedded.Base {
			return embeddedtest.NewInMemory()
		}),
		embeddedtest.BaseOnPing("succeeds on a fresh subject", func(
			tb testing.TB, subject embedded.Base,
		) {
			tb.Helper()
			testkit.NoError(tb, subject.Ping(tb.Context()), "an open subject answers a ping")
		}),
	)
}

// Declining the double for Base, which is a separate decision from dropping a
// check and is made per contract rather than per suite.
func TestBaseContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	embeddedtest.AssertBaseContract(t,
		embeddedtest.BaseSubject("in-memory", func() embedded.Base {
			return embeddedtest.NewInMemory()
		}),
		embeddedtest.BaseWithout("Ping/smoke"),
		embeddedtest.BaseWithoutDouble(),
	)
}

func TestCloserContract(t *testing.T) {
	t.Parallel()

	embeddedtest.AssertCloserContract(t,
		embeddedtest.CloserSubject("in-memory", func() embedded.Closer {
			return embeddedtest.NewInMemory()
		}),
		embeddedtest.CloserOnClose("is idempotent", func(
			tb testing.TB, subject embedded.Closer,
		) {
			tb.Helper()
			testkit.NoError(tb, subject.Close(tb.Context()), "the first close succeeds")
			testkit.NoError(tb, subject.Close(tb.Context()), "and so does the second")
		}),
	)
}

// Declining the double for Closer.
func TestCloserContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	embeddedtest.AssertCloserContract(t,
		embeddedtest.CloserSubject("in-memory", func() embedded.Closer {
			return embeddedtest.NewInMemory()
		}),
		embeddedtest.CloserWithout("Close/smoke"),
		embeddedtest.CloserWithoutDouble(),
	)
}

// Declining the double is separate from dropping a check, and a consumer who
// does not use the double should not pay for a second pass over every check.
func TestComposedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	embeddedtest.AssertComposedContract(t,
		embeddedtest.ComposedSubject("seeded", func() embedded.Composed {
			return embeddedtest.NewInMemory()
		}),
		embeddedtest.ComposedSeed(func(_ context.Context, subject embedded.Composed) error {
			// The fixture declares no writer, so nothing is derived and the
			// reader's hit path is unreachable without this. The key comes from
			// the fixture rather than being written out, so the seed and the
			// check cannot disagree about which identifier was stored.
			subject.(*embeddedtest.InMemory).Put(embeddedtest.DefaultComposedFixture().Key, "seeded")
			return nil
		}),
		embeddedtest.ComposedOnGet("returns what was seeded", func(
			tb testing.TB, subject embedded.Composed, id string,
		) {
			tb.Helper()
			got, err := subject.Get(tb.Context(), id)
			testkit.NoError(tb, err, "a seeded identifier is found")
			testkit.Equal(tb, got, "seeded", "and carries what was written")
		}),
		embeddedtest.ComposedWithout("Get/an error carries the zero value"),
		embeddedtest.ComposedWithoutDouble(),
	)
}
