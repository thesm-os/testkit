// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/core/tiers"
)

// TestEverySiblingReferenceIsClassified holds the partner tables total and
// disjoint over the live registry.
//
// The unclassified default is excluded, which keeps the pair synchronized —
// but a verdict reached by default is a judgment nobody made. A mixin landing
// upstream with a sibling parameter must be classified in the same build,
// and this is what asks.
//
// Disjoint as well as total, which it did not used to be. `scheduled.schedule`
// and `scheduled.fired` sat in both tables for a release: the driven rows won
// on `if` order, their exclusion reasons were never read, and by the time
// anyone read them they described a clock the clocked mode had made moot. An
// at-least-one census answered "classified" to all of it.
func TestEverySiblingReferenceIsClassified(t *testing.T) {
	t.Parallel()

	for _, m := range mixins.All() {
		for _, p := range m.Params {
			if p.Kind != shape.KindCallable {
				continue
			}
			testkit.True(t, tiers.PartnerClassified(m.Name, p.Key),
				m.Name+"."+p.Key+" carries a drive-or-exclude verdict, and only one")
		}
	}
}

// The scheduled siblings are driven, and only driven.
//
// The pair this item is about. Both rows sat in both tables for a release; the
// exclusion side lost on `if` order and its reasons went stale unread, and
// excluding them would have left the fixture with no action at all — the model
// generator refuses an interface whose sequences would drive nothing, so the
// exclusion would have deleted the tier that states the scheduling claim.
func TestScheduledSiblingsAreDriven(t *testing.T) {
	t.Parallel()

	for _, param := range []string{"schedule", "fired"} {
		testkit.True(t, tiers.PartnerClassified("scheduled", param),
			"scheduled."+param+" carries exactly one verdict")
		driven, reason := tiers.PartnerDriven("scheduled", param)
		testkit.True(t, driven, "scheduled."+param+" stays in the sequences")
		testkit.Equal(t, reason, "", "with no exclusion reason left to print")
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
