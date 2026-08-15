// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
)

// repoRoot is the tree the vacuity walk covers, relative to this module.
//
// The whole repository rather than the corpus alone. The generated output is
// where the class was found eleven times, but the assertion library, the
// generators and the laws are all written in the same idiom and all shipped by
// the same release — and a detector pointed only at generated code would say
// nothing about the hand-written half that generates it.
const repoRoot = "../.."

// vacuityOnce memoizes the walk, which parses several thousand files.
//
//nolint:gochecknoglobals // memoized measurement, test-only.
var vacuityOnce = sync.OnceValues(func() ([]gate.Vacuity, error) {
	return gate.Vacuities(repoRoot)
})

// Every assertion that cannot fail is registered, with the reason it survives.
//
// The detector this programme has been earning. Eleven items found this class
// by reading — a needle that was a substring of what the test configured, a
// builder seed comparing zero to itself, a guard asserting only that something
// failed, a law whose every arm returned nil. Each was fixed where it was
// found and nothing stopped the twelfth.
//
// Four rules, and the fourth is the one 1.3 asked for in an honest form: that
// item proposed grepping law files for refusal-shaped comments, which tests the
// comment. [gate.RuleUnfailableCheck] reads the returns instead, and it finds
// nothing in `engine/model/law` — which is the first evidence that 1.3's
// conversions hold, as opposed to having been done.
func TestEveryVacuityIsRegistered(t *testing.T) {
	t.Parallel()

	all, err := vacuityOnce()
	testkit.NoError(t, err, "the vacuity walk runs")

	_, unregistered := gate.VacuityCounts(all)
	lines := make([]string, 0, len(unregistered))
	for _, v := range unregistered {
		lines = append(lines, v.String())
	}
	testkit.True(t, len(unregistered) == 0,
		"every unfailable assertion is registered in VacuityDebt — unregistered:\n"+
			strings.Join(lines, "\n"))
}

// The register only ratchets down.
//
// Both directions, because a ceiling that is merely an upper bound stops being
// a measurement the moment the count drops: the next regression climbs back
// into the slack and nothing reports it. A class that shrank has to say so in
// the same commit that shrank it.
func TestEveryVacuityRowMatchesItsCeiling(t *testing.T) {
	t.Parallel()

	all, err := vacuityOnce()
	testkit.NoError(t, err, "the vacuity walk runs")

	counts, _ := gate.VacuityCounts(all)
	for key, row := range gate.VacuityDebt {
		got := counts[key]
		switch {
		case got > row.Ceiling:
			t.Errorf("%s: %d findings against a ceiling of %d — a new one landed in a "+
				"registered class, which the row's own reason may not cover", key, got, row.Ceiling)
		case got < row.Ceiling:
			t.Errorf("%s: %d findings against a ceiling of %d — lower the ceiling to %s "+
				"and bank the progress", key, got, row.Ceiling, strconv.Itoa(got))
		}
	}
}

// No row survives its class.
//
// A register row for a class nothing matches is an excuse for a defect that no
// longer exists, and it reads to the next person as a considered judgment
// about the current tree.
func TestNoVacuityRowIsStale(t *testing.T) {
	t.Parallel()

	all, err := vacuityOnce()
	testkit.NoError(t, err, "the vacuity walk runs")

	counts, _ := gate.VacuityCounts(all)
	for key := range gate.VacuityDebt {
		testkit.True(t, counts[key] > 0,
			key+" still matches something; delete the row if the class is gone")
	}
}

// Every row says what it is and what would close it.
//
// The bar the other three registers in this package hold their reasons to. Two
// of these rows are debt with a named fix and three are correct forever, and a
// reader has to be able to tell which without running the detector themselves.
func TestEveryVacuityRowArguesItsCase(t *testing.T) {
	t.Parallel()

	for key, row := range gate.VacuityDebt {
		testkit.True(t, len(row.Why) > 80,
			key+"'s row says what the class is and what closes it")
		rule, _, split := strings.Cut(key, " ")
		testkit.True(t, split && rule != "", key+" is keyed as `<rule> <path prefix>`")
	}
}

// The laws themselves reach a verdict.
//
// Called out separately from the register because it is the one class whose
// count must be zero rather than argued. A law that can only answer `nil` or
// `Vacuous` reports success on every input, and the run that carries it counts
// a law that was never able to disagree — which is what the saturation prover
// measures dynamically and what this catches without running anything.
func TestNoShippedLawIsUnfailable(t *testing.T) {
	t.Parallel()

	all, err := vacuityOnce()
	testkit.NoError(t, err, "the vacuity walk runs")

	var unfailable []string
	for _, v := range all {
		if v.Rule == gate.RuleUnfailableCheck {
			unfailable = append(unfailable, v.String())
		}
	}
	testkit.True(t, len(unfailable) == 0,
		"no shipped law answers a pass on every path:\n"+strings.Join(unfailable, "\n"))
}

// Each rule fires on the shape it names and holds off the shape it does not.
//
// The detector's own falsification. A rule that never fires is indistinguishable
// from a clean tree, which is the exact confusion the detector exists to end —
// so each is driven against source written to trip it, and against the nearest
// correct source that must not.
func TestEachRuleFiresOnItsOwnShape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		rule string
		src  string
		want bool
	}{{
		name: "self-comparison on two identical operands",
		rule: gate.RuleSelfComparison,
		src:  `func T(t *testing.T) { testkit.Equal(t, cfg.Retries, cfg.Retries, "m") }`,
		want: true,
	}, {
		name: "self-comparison spares two identical calls",
		// An idempotence claim reads as a self-comparison and is the check
		// most worth keeping: two calls, compared on purpose.
		rule: gate.RuleSelfComparison,
		src:  `func T(t *testing.T) { testkit.Equal(t, s.Get(k), s.Get(k), "m") }`,
		want: false,
	}, {
		name: "zero-expectation on a var handed to the subject",
		rule: gate.RuleZeroExpectation,
		src:  `func T(t *testing.T) { var a string; s.Put(a); testkit.Equal(t, got.A, a, "m") }`,
		want: true,
	}, {
		name: "zero-expectation spares a var the body writes to",
		rule: gate.RuleZeroExpectation,
		src:  `func T(t *testing.T) { var a string; a = s.Put(a); testkit.Equal(t, got.A, a, "m") }`,
		want: false,
	}, {
		name: "zero-expectation spares a standing zero nothing was given",
		// `var zero T` compared against a result says "this answered nothing",
		// which is a real claim and the shape 1.5's suppressions take.
		rule: gate.RuleZeroExpectation,
		src:  `func T(t *testing.T) { var zero string; testkit.Equal(t, got.A, zero, "m") }`,
		want: false,
	}, {
		name: "configured-needle on a substring the body supplied",
		rule: gate.RuleConfiguredNeedle,
		src:  `func T(t *testing.T) { s := New(WithName("alpha-1")); testkit.Contains(t, s.Msg(), "alpha", "m") }`,
		want: true,
	}, {
		name: "configured-needle spares a needle nothing configured",
		rule: gate.RuleConfiguredNeedle,
		src:  `func T(t *testing.T) { s := New(WithName("alpha-1")); testkit.Contains(t, s.Msg(), "refused", "m") }`,
		want: false,
	}, {
		name: "unchecked-rejection on a discarded answer",
		rule: gate.RuleUncheckedRejection,
		src:  `func T(t *testing.T) { testkit.Rejects(t, "r", func(tb testing.TB) {}) }`,
		want: true,
	}, {
		name: "unchecked-rejection spares one held to a phrase",
		rule: gate.RuleUncheckedRejection,
		src: `func T(t *testing.T) { got := testkit.Rejects(t, "r", func(tb testing.TB) {}); ` +
			`testkit.Assert(t, got).Contains("Put must be refused", "m") }`,
		want: false,
	}, {
		name: "unfailable-check on a law that only passes",
		rule: gate.RuleUnfailableCheck,
		src:  `func (l L) Check(rt *rapid.T, a, b T) error { if a == b { return nil }; return law.Vacuous }`,
		want: true,
	}, {
		name: "unfailable-check spares a law that can disagree",
		rule: gate.RuleUnfailableCheck,
		src:  `func (l L) Check(rt *rapid.T, a, b T) error { if a == b { return nil }; return errDiffer }`,
		want: false,
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, fires(t, c.src, c.rule), c.want,
				"the rule reports this shape exactly when it should")
		})
	}
}

// fires reports whether src trips the named rule.
func fires(t *testing.T, src, rule string) bool {
	t.Helper()

	dir := t.TempDir()
	testkit.NoError(t, writeGo(dir, "probe.go", "package p\n\n"+src+"\n"), "the probe writes")
	found, err := gate.Vacuities(dir)
	testkit.NoError(t, err, "the probe parses")

	for _, v := range found {
		if v.Rule == rule {
			return true
		}
	}
	return false
}

// writeGo puts one source file in dir.
func writeGo(dir, name, src string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600)
}

// A walk that cannot run is reported, never measured as clean.
//
// The arm that decides whether a red detector can be silenced by pointing it
// at nothing. An empty result read as "no vacuities" would make the whole
// register deletable by typo, which is the failure mode of every gate in this
// package and the reason each one tests this.
func TestVacuitiesSurfacesAWalkFailure(t *testing.T) {
	t.Parallel()

	t.Run("a root that is not there", func(t *testing.T) {
		t.Parallel()
		_, err := gate.Vacuities("definitely-not-a-directory")
		testkit.True(t, err != nil, "a missing root is an error, not an empty measurement")
	})

	t.Run("a file that does not parse", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testkit.NoError(t, writeGo(dir, "broken.go", "package p\nfunc ("), "the probe writes")
		_, err := gate.Vacuities(dir)
		testkit.True(t, err != nil, "source the parser refuses is reported, not skipped")
	})
}

// A rejection assigned to the blank identifier is discarded twice over.
//
// Its own case because it reads as a caller who meant to keep the message: the
// assignment is written and the value thrown away, which is the shape a
// half-finished edit leaves behind.
func TestUncheckedRejectionCatchesTheBlankAssignment(t *testing.T) {
	t.Parallel()

	src := `func T(t *testing.T) { _ = testkit.Rejects(t, "r", func(tb testing.TB) {}) }`
	testkit.True(t, fires(t, src, gate.RuleUncheckedRejection),
		"assigning the message to _ discards it as surely as never binding it")
}

// A finding in no registered class is handed back rather than counted.
//
// The half [TestEveryVacuityIsRegistered] cannot show while the tree is clean:
// that test passes by finding nothing, so the path that reports something is
// exercised here, on a finding no row can match.
func TestVacuityCountsReportsTheUnregistered(t *testing.T) {
	t.Parallel()

	found := []gate.Vacuity{
		{Rule: gate.RuleSelfComparison, File: "assert_test.go", Line: 1, Detail: "registered"},
		{Rule: gate.RuleSelfComparison, File: "somewhere/else.go", Line: 2, Detail: "not"},
	}
	counts, unregistered := gate.VacuityCounts(found)

	testkit.Equal(t, counts["self-comparison assert_test.go"], 1, "the registered one is counted")
	testkit.Len(t, unregistered, 1, "the other is handed back")
	testkit.Contains(t, unregistered[0].String(), "somewhere/else.go:2",
		"and it renders with the file and line a reader needs")
}

// Findings sort by file, then by line.
//
// A gate that prints its findings in map order prints them differently every
// run, and a reader comparing two runs cannot tell a new finding from a
// reordering.
func TestVacuitiesSortStably(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	body := `func T(t *testing.T) { testkit.Equal(t, a, a, "m"); testkit.Equal(t, b, b, "m") }`
	testkit.NoError(t, writeGo(dir, "b.go", "package p\n\n"+body+"\n"), "the second file writes")
	testkit.NoError(t, writeGo(dir, "a.go", "package p\n\n"+body+"\n"), "the first file writes")

	found, err := gate.Vacuities(dir)
	testkit.NoError(t, err, "the probe parses")
	testkit.Len(t, found, 4, "two findings per file")

	order := make([]string, 0, len(found))
	for _, v := range found {
		order = append(order, v.File+":"+strconv.Itoa(v.Line))
	}
	testkit.Equal(t, strings.Join(order, " "), "a.go:3 a.go:3 b.go:3 b.go:3",
		"file first, then line")
}
