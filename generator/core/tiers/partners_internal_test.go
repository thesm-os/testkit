// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
)

// partnerVerdict answers for all four combinations of the two lookups, and the
// fourth is the one this test exists for.
//
// A pair in both tables cannot be reached through [PartnerDriven] — the census
// fails a build where one exists — so the arm that handles it would be dead
// code checked by nothing. That is the same shape as the defect it was written
// to answer: two rows that satisfied every question anyone asked and stated a
// reason nothing read.
func TestPartnerVerdict(t *testing.T) {
	t.Parallel()

	t.Run("driven alone drives", func(t *testing.T) {
		t.Parallel()
		driven, reason := partnerVerdict(true, false, "")
		testkit.True(t, driven, "a role reference stays in the sequences")
		testkit.Equal(t, reason, "", "with nothing to explain")
	})

	t.Run("excluded alone carries its own reason", func(t *testing.T) {
		t.Parallel()
		driven, reason := partnerVerdict(false, true, "a validator")
		testkit.False(t, driven, "an override is kept out")
		testkit.Equal(t, reason, "a validator", "and the header prints the table's words")
	})

	t.Run("neither is the unclassified default", func(t *testing.T) {
		t.Parallel()
		driven, reason := partnerVerdict(false, false, "")
		testkit.False(t, driven, "the safe side")
		testkit.Assert(t, reason).Contains("unclassified", "named as the default it is")
	})

	t.Run("both is a defect, and says so instead of picking", func(t *testing.T) {
		t.Parallel()
		// The excluded reason is deliberately non-empty here: a conflict must
		// not print it. It is the reason for a verdict nobody settled, and
		// printing it would tell a reader the exclusion won on its merits.
		driven, reason := partnerVerdict(true, true, "a validator")
		testkit.False(t, driven, "neither table wins, and excluded is the safe side")
		testkit.Assert(t, reason).Contains("both driven and excluded",
			"the reason names the disagreement rather than one side of it")
		testkit.False(t, strings.Contains(reason, "a validator"),
			"and not the losing table's own words, which would read as a settled verdict")
	})
}
