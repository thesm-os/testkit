// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package snapshotisolationtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/snapshotisolation"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/snapshotisolation/snapshotisolationtest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	snapshotisolationtest.RunMixed(
		t,
		// The anomaly doors stay unarmed here, deliberately: this subject is
		// a passive log, and the entries the property records are drawn from
		// pools — a fabricated history whose "anomalies" are the draws'
		// collisions, not any subject's interleaving. Arming History proved
		// exactly that on its first run. The doors are generated and typed;
		// a transactional subject whose History reports its own commits is
		// what arms them honestly.
		snapshotisolationtest.MixedHarness[*snapshotisolationtest.InMemory]{
			Name: "in-memory",
			New:  snapshotisolationtest.NewInMemory,
		},
		snapshotisolationtest.MixedChecks{
			{
				Method: "History",
				Name:   "hands-back-a-copy",
				Claim:  "History hands back a copy, not the backing array",
				Run: func(tb testing.TB, s snapshotisolation.Mixed, fx snapshotisolationtest.MixedFixture) {
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
