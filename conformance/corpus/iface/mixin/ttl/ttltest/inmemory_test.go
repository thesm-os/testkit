// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ttltest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttl"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/ttl/ttltest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	ttltest.AssertMixedContract(t,
		ttltest.MixedSubject("in-memory", func() ttl.Mixed {
			return ttltest.NewInMemory()
		}),
		ttltest.MixedOnRead("reports the declared sentinel for an absent key", func(
			tb testing.TB, subject ttl.Mixed, key string,
		) {
			tb.Helper()
			// The suite seeds through Put, so the seeded key is live on a
			// clock nothing advanced. An unwritten key reports the sentinel
			// the directive names — which is what a lapsed read reports too,
			// so a caller cannot tell "never stored" from "stored and gone".
			got, err := subject.Read(tb.Context(), key)
			testkit.NoError(tb, err, "the seeded key is within its lifetime")
			testkit.Equal(tb, got.Key, key, "and answers under the key it was stored with")

			_, err = subject.Read(tb.Context(), key+"-absent")
			testkit.ErrorIs(tb, err, ttl.ErrExpired, "an absent key reports the declared sentinel")

			// The lever: an entry stamped a lifetime ago is exactly what an
			// elapsed one looks like, and the expiry arm is the whole claim
			// the directive makes.
			testkit.NoError(tb, subject.Put(tb.Context(),
				ttl.Value{Key: ttltest.StaleKey, Body: "elapsed"}), "the entry stores")
			_, err = subject.Read(tb.Context(), ttltest.StaleKey)
			testkit.ErrorIs(tb, err, ttl.ErrExpired, "and a lapsed read reports the sentinel")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	ttltest.AssertMixedContract(t,
		ttltest.MixedSubject("in-memory", func() ttl.Mixed {
			return ttltest.NewInMemory()
		}),
		ttltest.MixedWithoutDouble(),
	)
}
