// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ttltest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttl"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttl/ttltest"
)

// `//testkit:mixin ttl notfound=ErrExpired` is what makes Read/miss derivable:
// the declaration names what a read owes for a key nothing holds, and expiry
// and absence report alike here, so one sentinel covers both.
//
// The lapse itself is the model tier's — it needs a clock the run advances.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	ttltest.RunMixed(t,
		ttltest.MixedHarness[*ttltest.InMemory]{Name: "in-memory", New: ttltest.NewInMemory},
		ttltest.MixedChecks{
			{
				Method: "Read",
				Name:   "lapsed-reads-report-the-sentinel",
				Claim:  "Read reports the declared sentinel for an entry whose lifetime has passed",
				Run: func(tb testing.TB, s ttl.Mixed, fx ttltest.MixedFixture) {
					tb.Helper()
					// A live entry first, so the sentinel below is about the
					// lapse rather than about an empty store.
					written := fx.Value()
					testkit.NoError(tb, s.Put(tb.Context(), written), "the entry stores")

					got, err := s.Read(tb.Context(), written.Key)
					testkit.NoError(tb, err, "and is within its lifetime")
					testkit.Equal(tb, got.Key, written.Key,
						"answering under the key it was stored with")

					// The lever: an entry stamped a lifetime ago is exactly
					// what an elapsed one looks like, and the expiry arm is the
					// whole claim the directive makes. Read/miss covers the
					// key nothing wrote; this covers the one that lapsed.
					testkit.NoError(tb, s.Put(tb.Context(),
						ttl.Value{Key: ttltest.StaleKey, Body: "elapsed"}), "the stale entry stores")
					_, err = s.Read(tb.Context(), ttltest.StaleKey)
					testkit.ErrorIs(tb, err, ttl.ErrExpired, "and a lapsed read reports the sentinel")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	ttltest.RunMixed(t,
		ttltest.MixedHarness[*ttltest.InMemory]{Name: "in-memory", New: ttltest.NewInMemory},
		ttltest.MixedSuite.Without(ttltest.MixedSuite.Checks.Put.Smoke()),
	)
}
