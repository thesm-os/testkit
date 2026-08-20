// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package serializabletest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/serializable"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/serializable/serializabletest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	serializabletest.RunMixed(t,
		// The anomaly door stays unarmed here for the reason its snapshot
		// sibling's does: this subject is a passive log, and the entries the
		// property records are drawn from pools — a fabricated history whose
		// "anomalies" are the draws' collisions rather than any subject's
		// interleaving. The door is generated and typed; a transactional
		// subject whose History reports its own commits is what arms it.
		serializabletest.MixedHarness[*serializabletest.InMemory]{Name: "in-memory", New: serializabletest.NewInMemory},
		serializabletest.MixedChecks{
			{
				Method: "History",
				Name:   "hands-back-a-copy",
				Claim:  "History hands back a copy, not the backing array",
				Run: func(tb testing.TB, s serializable.Mixed, fx serializabletest.MixedFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Record(tb.Context(), fx.Entry()), "an entry is recorded")

					// Mutating what History returned must not reach the
					// subject: an anomaly check walks this while the subject
					// may still record.
					got, err := s.History(tb.Context())
					testkit.NoError(tb, err, "the history is readable")
					testkit.Equal(tb, len(got), 1, "the recorded entry is there")

					got[0].Txn = 99
					again, err := s.History(tb.Context())
					testkit.NoError(tb, err, "the history is still readable")
					testkit.NotEqual(tb, again[0].Txn, 99, "the caller's edit did not reach the subject")
				},
			},
		},
	)
}
