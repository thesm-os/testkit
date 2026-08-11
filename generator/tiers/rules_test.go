// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/tiers"
)

// TestSelectRequiresEveryClassification holds the AND in [tiers.Rule.Needs].
//
// The conjunction is what keeps `{writer, idempotent}` and
// `{lifecycle, idempotent}` from contesting one method with no tiebreak
// written down. If Needs were satisfied by any member rather than all, a
// lifecycle carrying `idempotent` would bind the write law too — and that law
// would call a writer the method is not, against a reader the interface may
// not have.
func TestSelectRequiresEveryClassification(t *testing.T) {
	t.Parallel()

	laws := func(classifications ...string) []string {
		out := make([]string, 0, len(tiers.Rules()))
		for _, r := range tiers.Select(classifications, nil) {
			out = append(out, r.Law)
		}
		return out
	}

	t.Run("a partial set selects nothing needing the rest", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, slices.Contains(laws("idempotent"), lawid.IdempotentWrite),
			"idempotent alone does not say which shape it decorates")
		testkit.False(t, slices.Contains(laws("idempotent"), lawid.IdempotentLifecycle),
			"nor does it select the lifecycle form")
	})

	t.Run("the full set selects the law", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, slices.Contains(laws("writer", "idempotent"), lawid.IdempotentWrite),
			"a writer carrying idempotent owes the write form")
		testkit.True(t, slices.Contains(laws("lifecycle", "idempotent"), lawid.IdempotentLifecycle),
			"a lifecycle carrying idempotent owes the lifecycle form")
	})

	t.Run("the detector shape a rule discriminates on is single-valued", func(t *testing.T) {
		t.Parallel()
		// The exclusivity of the two idempotent rules rests entirely on this:
		// a method carries one detector shape, so at most one of them can
		// match. [Select] has no precedence and no supersession, deliberately,
		// because with a single-valued shape none is needed.
		//
		// Asserted against eidos rather than assumed. If the shape key ever
		// became a list the way mixins are, both rules could match one method
		// and a tiebreak would have to be written — and this failing is how
		// that would be found, rather than a write law running against a
		// lifecycle method somewhere in a consumer's suite.
		bag := sdk.NewBag()
		shape.MetaShape.Set(bag, "writer", "tiers-test")
		shape.MetaShape.Set(bag, "lifecycle", "tiers-test")
		got, _ := shape.MetaShape.Get(bag)
		testkit.Equal(t, got, "lifecycle",
			"a second shape stamp replaces the first rather than joining it")

		// The contrast that makes the point: mixins accumulate, which is why
		// a rule may need several of them and only ever one shape.
		shape.MetaMixins.Set(bag, []string{"idempotent"}, "tiers-test")
		mixins, _ := shape.MetaMixins.Get(bag)
		testkit.Len(t, mixins, 1, "mixins are the plural axis")
	})
}

// TestSelectReadsParameterConditions covers the three forms of [tiers.Condition].
//
// Each exists for a law the classification alone cannot choose. Getting the
// absent case wrong is the costly one: `codec` defaults to exact fidelity, so
// a rule keyed on the stamp being present would silently unbind the roundtrip
// law for everyone who relied on the default.
func TestSelectReadsParameterConditions(t *testing.T) {
	t.Parallel()

	laws := func(classification string, params map[string]string) []string {
		out := make([]string, 0, len(tiers.Rules()))
		for _, r := range tiers.Select([]string{classification}, params) {
			out = append(out, r.Law)
		}
		return out
	}

	t.Run("an equality selects one of a family", func(t *testing.T) {
		t.Parallel()
		got := laws("publisher", map[string]string{
			shape.ContractParamKey("publisher", "mode").Name(): "exactly-once",
		})
		testkit.True(t, slices.Contains(got, lawid.PublisherExactlyOnce), "the declared mode binds")
		testkit.False(t, slices.Contains(got, lawid.PublisherAtLeastOnce), "and only that one")
	})

	t.Run("an unstamped parameter selects no refined form", func(t *testing.T) {
		t.Parallel()
		got := laws("publisher", nil)
		testkit.True(t, slices.Contains(got, lawid.PublisherDelivers),
			"delivery is claimed by the contract itself")
		for _, refined := range []string{
			lawid.PublisherAtLeastOnce, lawid.PublisherAtMostOnce, lawid.PublisherExactlyOnce,
		} {
			testkit.False(t, slices.Contains(got, refined),
				"an absent mode is unstated, not a default")
		}
	})

	t.Run("not-equals holds for an absent stamp and for another value", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, slices.Contains(laws("codec", nil), lawid.Roundtrip),
			"exact fidelity is the documented default")
		testkit.True(t, slices.Contains(
			laws("codec", map[string]string{shape.ContractParamKey("codec", "fidelity").Name(): "exact"}),
			lawid.Roundtrip,
		), "and writing the default out changes nothing")
		testkit.False(t, slices.Contains(
			laws("codec", map[string]string{shape.ContractParamKey("codec", "fidelity").Name(): "lossy"}),
			lawid.Roundtrip,
		), "a lossy codec is not held to the identity")
		testkit.True(t, slices.Contains(
			laws("codec", map[string]string{shape.ContractParamKey("codec", "fidelity").Name(): "lossy"}),
			lawid.LossyRoundtrip,
		), "it is held to the weaker claim instead")
	})
}

// TestSelectIsEmptyForAnUnclassifiedMethod holds the floor.
//
// A method nothing classified owes nothing, and a rule with an empty Needs
// would bind to every method in every interface — which is the shape of defect
// that only shows up as a law failing somewhere it was never meant to run.
func TestSelectIsEmptyForAnUnclassifiedMethod(t *testing.T) {
	t.Parallel()

	testkit.Len(t, tiers.Select(nil, nil), 0, "no classifications, no laws")
	testkit.Len(t, tiers.Select([]string{"not-a-classification"}, nil), 0,
		"an unknown classification selects nothing")

	for _, r := range tiers.Rules() {
		testkit.True(t, len(r.Needs) > 0, r.Law+" names what selects it")
	}
}

// TestLawsForReportsUnderEveryClassificationThatReachesIt covers the census
// side, which reads differently from the binding side.
//
// A rule needing two classifications is reachable from either, so both must
// report it — a header that listed AUTO-IDEMPOTENT-LIFECYCLE under `lifecycle`
// but not under `idempotent` would tell a reader their mixin was ignored.
func TestLawsForReportsUnderEveryClassificationThatReachesIt(t *testing.T) {
	t.Parallel()

	for _, c := range []string{"idempotent", "lifecycle"} {
		testkit.True(t, slices.Contains(tiers.LawsFor(c), lawid.IdempotentLifecycle),
			c+" reaches the lifecycle form")
	}
	testkit.True(t, slices.IsSorted(tiers.LawsFor("cursor")), "the report is sorted")
	testkit.Len(t, tiers.LawsFor("reader"), 0,
		"a classification the suite tier checks earns no law here")
}

// TestRulesHandsOutACopy stops a caller from editing the catalogue.
//
// It is a package-level slice read by three modules; a consumer that sorted or
// filtered it in place would change what every later reader selects, and the
// symptom would be a law that binds in one run and not the next.
func TestRulesHandsOutACopy(t *testing.T) {
	t.Parallel()

	first := tiers.Rules()
	testkit.True(t, len(first) > 0, "the catalogue is populated")
	first[0] = tiers.Rule{Law: "AUTO-TAMPERED"}

	testkit.NotEqual(t, tiers.Rules()[0].Law, "AUTO-TAMPERED",
		"the catalogue is unchanged by a caller's edit")
}
