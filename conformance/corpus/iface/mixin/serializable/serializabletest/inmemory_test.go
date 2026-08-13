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

	serializabletest.AssertMixedContract(t,
		// The anomaly door stays unarmed here for the reason its snapshot
		// sibling's does: this subject is a passive log, and the entries the
		// property records are drawn from pools — a fabricated history whose
		// "anomalies" are the draws' collisions rather than any subject's
		// interleaving. The door is generated and typed; a transactional
		// subject whose History reports its own commits is what arms it.
		serializabletest.MixedModel(),
		serializabletest.MixedSubject("in-memory", func() serializable.Mixed {
			return serializabletest.NewInMemory()
		}),
		serializabletest.MixedOnHistory("hands back a copy, not the backing array", func(
			tb testing.TB, subject serializable.Mixed,
		) {
			tb.Helper()
			// The suite seeds through Record, so the history is non-empty.
			// Mutating what History returned must not reach the subject: an
			// anomaly check walks this while the subject may still record.
			got, err := subject.History(tb.Context())
			testkit.NoError(tb, err, "the history is readable")
			testkit.Equal(tb, len(got), 1, "the seeded entry is there")

			got[0].Txn = 99
			again, err := subject.History(tb.Context())
			testkit.NoError(tb, err, "the history is still readable")
			testkit.NotEqual(tb, again[0].Txn, 99, "the caller's edit did not reach the subject")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	serializabletest.AssertMixedContract(t,
		serializabletest.MixedSubject("in-memory", func() serializable.Mixed {
			return serializabletest.NewInMemory()
		}),
		serializabletest.MixedWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestMixedSaturation(t *testing.T) {
	t.Parallel()
	serializabletest.MixedModelSaturation(t, func() serializable.Mixed {
		return serializabletest.NewInMemory()
	})
}
