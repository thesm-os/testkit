// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package embeddedforeigntest_test

import (
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

	fx := embeddedforeigntest.DefaultStreamFixture()

	embeddedforeigntest.RunStream(t,
		embeddedforeigntest.StreamHarness[*embeddedforeigntest.InMemory]{
			Name: "in-memory",
			// Stream declares no writer, so the reader's hit path is
			// unreachable without a seeded constructor.
			New: func() *embeddedforeigntest.InMemory {
				s := embeddedforeigntest.NewInMemory()
				s.Put(fx.Key(), "streamed")
				return s
			},
		},
		embeddedforeigntest.StreamChecks{
			{
				Method: "Read",
				Name:   "returns-what-was-seeded",
				Claim:  "Read returns what was seeded",
				Run: func(tb testing.TB, s embeddedforeign.Stream, fx embeddedforeigntest.StreamFixture) {
					tb.Helper()
					got, err := s.Read(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a seeded key is found")
					testkit.Equal(tb, got, "streamed", "and carries what was written")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestStreamContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	embeddedforeigntest.RunStream(
		t,
		embeddedforeigntest.StreamHarness[*embeddedforeigntest.InMemory]{
			Name: "in-memory",
			New:  embeddedforeigntest.NewInMemory,
		},
		embeddedforeigntest.StreamSuite.Without(embeddedforeigntest.StreamSuite.Checks.Close.Smoke()),
	)
}
