// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package writesfollowreadstest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/writesfollowreads"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/writesfollowreads/writesfollowreadstest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	writesfollowreadstest.AssertMixedContract(t,
		writesfollowreadstest.MixedModel(),
		writesfollowreadstest.MixedSubject("in-memory", func() writesfollowreads.Mixed {
			return writesfollowreadstest.NewInMemory()
		}),
		writesfollowreadstest.MixedOnGet("returns what Store wrote", func(
			tb testing.TB, subject writesfollowreads.Mixed, key string,
		) {
			tb.Helper()
			// The suite seeds through Store, so the key is already present —
			// which is what makes this a statement about the pair rather than
			// about Get alone.
			got, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "the seeded key is present")
			testkit.Equal(tb, got.Key, key, "and Get answers under the key it was stored with")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	writesfollowreadstest.AssertMixedContract(t,
		writesfollowreadstest.MixedSubject("in-memory", func() writesfollowreads.Mixed {
			return writesfollowreadstest.NewInMemory()
		}),
		writesfollowreadstest.MixedWithoutDouble(),
	)
}
