// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/tiers"
)

// TestEverySiblingReferenceIsClassified holds the partner tables total over
// the live registry.
//
// The unclassified default is excluded, which keeps the pair synchronized —
// but a verdict reached by default is a judgment nobody made. A mixin landing
// upstream with a sibling parameter must be classified in the same build,
// and this is what asks.
func TestEverySiblingReferenceIsClassified(t *testing.T) {
	t.Parallel()

	for _, m := range mixins.All() {
		for _, p := range m.SiblingParams {
			testkit.True(t, tiers.PartnerClassified(m.Name, p),
				m.Name+"."+p+" carries a drive-or-exclude verdict")
		}
	}
}

// TestPartnerDriven pins the three answers: a role reference drives, an
// override is excluded with a reason a header can print, and the unclassified
// arm says what it is.
func TestPartnerDriven(t *testing.T) {
	t.Parallel()

	driven, reason := tiers.PartnerDriven("ttl", "put")
	testkit.True(t, driven, "a put that is a writer stays in the sequences")
	testkit.Equal(t, reason, "", "with nothing to explain")

	driven, reason = tiers.PartnerDriven("validates", "fn")
	testkit.False(t, driven, "a validator is not driven as the writer it resembles")
	testkit.Assert(t, reason).Contains("validator", "and the reason says why")

	driven, reason = tiers.PartnerDriven("not-a-mixin", "fn")
	testkit.False(t, driven, "the unclassified default is the safe side")
	testkit.Assert(t, reason).Contains("unclassified", "named as the default it is")
}
