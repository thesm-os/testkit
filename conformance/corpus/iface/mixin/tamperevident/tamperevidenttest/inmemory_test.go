// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tamperevidenttest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/tamperevident"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/tamperevident/tamperevidenttest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	tamperevidenttest.AssertMixedContract(t,
		tamperevidenttest.MixedModel(),
		tamperevidenttest.MixedSubject("in-memory", func() tamperevident.Mixed {
			return tamperevidenttest.NewInMemory()
		}),
		tamperevidenttest.MixedOnVerify("detects a value altered behind its back", func(
			tb testing.TB, subject tamperevident.Mixed,
		) {
			tb.Helper()
			// The suite seeds through Store, so there is something to alter.
			testkit.NoError(tb, subject.Verify(tb.Context()), "an untouched value verifies")
			testkit.NoError(tb, subject.Corrupt(tb.Context()), "the bytes are altered")
			testkit.Error(tb, subject.Verify(tb.Context()),
				"and the alteration is detected rather than served")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	tamperevidenttest.AssertMixedContract(t,
		tamperevidenttest.MixedSubject("in-memory", func() tamperevident.Mixed {
			return tamperevidenttest.NewInMemory()
		}),
		tamperevidenttest.MixedWithoutDouble(),
	)
}
