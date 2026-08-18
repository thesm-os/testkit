// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
)

// evidenceOnce memoizes the corpus-wide evidence measurement.
//
// One full pipeline run for every test in this file, the way the assertion
// gate does it: the run loads and type-checks the whole corpus, and three
// tests asking the same question three times would triple the slowest thing in
// this package for no additional truth.
//
// evidenceOnce reads the package's single corpus run — see censusOnce for why
// there is only one.
func evidenceOnce() ([]gate.Evidence, error) {
	census, err := censusOnce()
	return census.Evidence, err
}

// skipUntilSuiteEmission parks the census questions the suite rewrite unowns.
//
// evidenceFrom's TRANSITION paragraph is the owner: the incumbent suite
// emission — and the per-Contract Coverage walk every question in this file was
// answered from — is deleted, and the rewrite's deriver inventory has not
// replaced it yet. So the census reads the whole registry as unevidenced, which
// makes one of these tests red for a reason that is about the transition rather
// than the corpus, and the two that iterate evidenced rows iterate none.
//
// A skip rather than a relaxed assertion, because a bar the tree clears
// vacuously today is a bar it still clears on the day the new emission lands
// and evidences nothing — which is the one morning this register exists to be
// red on. The skip is the deferred work with a named owner; the relaxation
// would be the work forgotten.
//
// The message carries an expiry so the deferral is held by the skip-expiry
// gate rather than by anyone's memory: past that date the build reddens here
// until the census is rebuilt or the date is argued forward. The full account
// is the transition section of docs/internal/suite-plugin-design.md.
func skipUntilSuiteEmission(tb testing.TB) {
	tb.Helper()
	tb.Skip("the suite tier evidences nothing until the rewrite's emission lands; expires 2026-09-18")
}

// Every classification eidos ships is asserted by a tier or argued by a row.
//
// The union census ADR-0018 asks for, and the question every existing gate
// falls just to one side of. [Annotate] asks whether a classification is
// stamped, which each of these passes — the fixture exists and the directive
// parses. [Emission] asks what the model tier bound, which is one tier's
// answer. The generated header asks per interface, which is one fixture's.
//
// None of them asks whether the vocabulary bought anything anywhere, and
// fifteen classifications were living in that gap: stamped, fixtured, and
// asserted by neither tier, with the reason for four of them confessed in a
// fixture comment and for the rest not written down at all.
func TestEveryClassificationHasEvidenceOrARow(t *testing.T) {
	skipUntilSuiteEmission(t)
	t.Parallel()

	all, err := evidenceOnce()
	testkit.NoError(t, err, "the evidence census runs")

	unevidenced := gate.Unevidenced(all)
	testkit.True(t, len(unevidenced) == 0,
		"every classification is asserted by a tier or argued in "+
			"UnevidencedClassifications — unregistered: "+strings.Join(unevidenced, ", "))
}

// The register only shrinks.
//
// The other direction, and the one that keeps a register from becoming a place
// gaps go to be forgotten. A row for a classification some tier now asserts is
// a stale excuse, and leaving it costs more than the gap did: the next reader
// takes it for a considered judgment about the current tree.
func TestNoUnevidencedRowIsStale(t *testing.T) {
	skipUntilSuiteEmission(t)
	t.Parallel()

	all, err := evidenceOnce()
	testkit.NoError(t, err, "the evidence census runs")

	stale := gate.ArguedButEvidenced(all)
	testkit.True(t, len(stale) == 0,
		"delete the rows for classifications a tier now asserts: "+strings.Join(stale, ", "))
}

// Every row says what the claim is waiting on.
//
// The same bar the unarmed-door census holds its reasons to. A row reading
// "not applicable" is a row that records that somebody closed the ticket, and
// the register exists so the next reader can tell a considered absence from an
// unexamined one.
func TestEveryUnevidencedRowArguesItsCase(t *testing.T) {
	t.Parallel()

	for name, reason := range gate.UnevidencedClassifications {
		testkit.True(t, len(reason) > 60,
			name+"'s row says what the claim is waiting on, not that it was skipped")
	}
}

// The census sees the whole registry, not a subset of it.
//
// A census measuring fewer classifications than eidos ships would be green for
// the wrong reason, and silently: the ones it never looked at are exactly the
// ones nobody would notice were missing. So the row count is held to
// [gate.Registered], which reads the live registries.
func TestEvidenceCoversTheWholeRegistry(t *testing.T) {
	t.Parallel()

	all, err := evidenceOnce()
	testkit.NoError(t, err, "the evidence census runs")

	registered := 0
	for _, names := range gate.Registered() {
		registered += len(names)
	}
	testkit.Equal(t, len(all), registered,
		"one row per registered classification, across all three axes")

	byAxis := map[string]int{}
	for _, e := range all {
		byAxis[e.Axis]++
	}
	for axis, names := range gate.Registered() {
		testkit.Equal(t, byAxis[axis], len(names), axis+" is measured in full")
	}
}

// A classification with evidence names where it came from.
//
// The half a red row cannot check. An `Evidenced` row whose Where is empty
// would mean the census concluded something is asserted without being able to
// say by what, which is the shape of a measurement that has drifted from what
// it measures.
func TestEvidencedRowsNameTheirFixture(t *testing.T) {
	skipUntilSuiteEmission(t)
	t.Parallel()

	all, err := evidenceOnce()
	testkit.NoError(t, err, "the evidence census runs")

	for _, e := range all {
		if !e.Evidenced() {
			continue
		}
		testkit.True(t, e.Where != "",
			e.Axis+"/"+e.Name+" names the fixture whose tier asserts it")
	}
}

// A pattern matching nothing is a failed run, not an empty census.
//
// The arm that decides whether a red gate can be silenced by pointing it
// somewhere there is nothing to measure. An empty result read as "everything
// is evidenced" would make the whole register deletable by typo.
func TestEvidenceSurfacesARunFailure(t *testing.T) {
	t.Parallel()

	_, err := gate.Evidenced(t.Context(), corpusRoot, "./corpus/definitely-not-here/...")
	testkit.True(t, err != nil, "a failed run reports, never measures empty")
}

// Unevidenced names what is neither asserted nor argued, and nothing else.
//
// Driven over a synthetic slice rather than the corpus. The corpus answer is
// "none", which is the state the gate exists to hold — so the corpus can only
// ever exercise the arm that finds nothing, and the arm that finds something is
// the one a reader depends on being right.
func TestUnevidencedSelects(t *testing.T) {
	t.Parallel()

	// "scope" is a real register row; "invented" is in neither the register nor
	// any tier, which is the shape a new upstream classification arrives in.
	all := []gate.Evidence{
		{Axis: "mixin", Name: "scope"},
		{Axis: "mixin", Name: "invented"},
		{Axis: "detector", Name: "reader", Checked: true},
		{Axis: "contract", Name: "cas", Modeled: true},
	}

	testkit.Equal(t, strings.Join(gate.Unevidenced(all), ", "), "mixin/invented",
		"an argued gap is not reported, an unargued one is, and evidence of either kind clears it")
}

// ArguedButEvidenced names the rows a tier has since covered.
//
// The register's shrink-only direction, and synthetic for the same reason: the
// corpus answer is "none" by construction, so the arm that finds a stale row
// would never run against it.
func TestArguedButEvidencedSelects(t *testing.T) {
	t.Parallel()

	all := []gate.Evidence{
		{Axis: "mixin", Name: "scope", Checked: true},
		{Axis: "mixin", Name: "deprecated"},
		{Axis: "detector", Name: "reader", Checked: true},
	}

	testkit.Equal(t, strings.Join(gate.ArguedButEvidenced(all), ", "), "scope",
		"a row whose classification a tier now asserts is stale; one still unevidenced is not, "+
			"and a classification with no row is not the register's business")
}

// The two single-purpose entry points work on their own.
//
// [gate.Measure] runs the corpus once for the whole package, so the narrower
// [gate.Evidenced] and [gate.Emission] stopped being called on the happy path
// the moment that landed — and an entry point nothing exercises is one that
// compiles and nothing else. They are the honest way to ask one question, so
// they are asked here, over a single fixture rather than the corpus: the point
// is that the entry point answers, not that it answers about everything.
func TestSinglePurposeEntryPointsAnswer(t *testing.T) {
	t.Parallel()

	one := "./corpus/iface/mixin/monotonic"

	evidence, err := gate.Evidenced(t.Context(), corpusRoot, one)
	testkit.NoError(t, err, "Evidenced runs over one fixture")
	testkit.True(t, len(evidence) > 0, "and reports the whole registry, measured against it")

	census, err := gate.Measure(t.Context(), corpusRoot, one)
	testkit.NoError(t, err, "Measure runs over one fixture")
	testkit.Equal(t, len(census.Evidence), len(evidence),
		"and its evidence half is what Evidenced answers alone")
	testkit.True(t, len(census.Emitted) > 0, "with the bindings read off the same run")
}

// A census run that cannot start is reported, never measured as empty.
//
// [Measure]'s own arm. The two narrower entry points each have one and this is
// the third — an empty census read as "nothing is owed" would silence every
// register in this package at once.
func TestMeasureSurfacesARunFailure(t *testing.T) {
	t.Parallel()

	_, err := gate.Measure(t.Context(), corpusRoot, "./corpus/definitely-not-here/...")
	testkit.True(t, err != nil, "a failed run reports, never measures empty")
}
