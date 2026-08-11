// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package embeddedforeigntest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/embeddedforeign"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/embeddedforeign/embeddedforeigntest"
)

// An embed from outside the run is still part of the contract.
//
// Close has no declaration in embeddedforeign — it arrives from io.Closer,
// projected off the type-checker — so a run that flattened only the source
// would hold an implementation to Read alone and report success. The proof is
// that the harness runs Close/smoke at all, which is why this file names no
// check for Close itself: the generated one is the claim.
func TestStreamContract(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced: nothing here
	// passes StreamWithFixture, so the derivation stands.
	fixture := embeddedforeigntest.DefaultStreamFixture()

	embeddedforeigntest.AssertStreamContract(t,
		embeddedforeigntest.StreamSubject("in-memory", func() embeddedforeign.Stream {
			return embeddedforeigntest.NewInMemory()
		}),
		embeddedforeigntest.StreamSeed(func(_ context.Context, subject embeddedforeign.Stream) error {
			// Stream declares no writer, so nothing is derived and the reader's
			// hit path is unreachable without this. A seed may reach for the
			// concrete subject: it runs before the double wraps it and sees
			// what the factory made. A check may not.
			subject.(*embeddedforeigntest.InMemory).Put(fixture.Key, "streamed")
			return nil
		}),
		embeddedforeigntest.StreamOnRead("returns what was seeded", func(
			tb testing.TB, subject embeddedforeign.Stream, key string,
		) {
			tb.Helper()
			got, err := subject.Read(tb.Context(), key)
			testkit.NoError(tb, err, "a seeded key is found")
			testkit.Equal(tb, got, "streamed", "and carries what was written")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestStreamContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	embeddedforeigntest.AssertStreamContract(t,
		embeddedforeigntest.StreamSubject("in-memory", func() embeddedforeign.Stream {
			return embeddedforeigntest.NewInMemory()
		}),
		embeddedforeigntest.StreamWithout("Close/smoke"),
		embeddedforeigntest.StreamWithoutDouble(),
	)
}
