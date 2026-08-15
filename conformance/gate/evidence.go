// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"cmp"
	"context"
	"maps"
	"slices"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/suite"
)

// Evidence is one classification and what, if anything, asserts it.
//
// The union census ADR-0018 asks for. [Annotate] measures that a
// classification is *stamped* somewhere in the corpus, which every one of
// these passes — a fixture exists, its directive parses, the annotator
// produces the mark. What that cannot see is whether the stamp bought
// anything: a classification can be registered upstream, fixtured here, and
// asserted by neither tier, and every existing gate stays green because each
// is asking a question this one falls between.
//
// So the two tiers are asked directly, unioned across the whole corpus rather
// than per fixture. Per fixture is the generated header's question and it has
// a different answer — a classification checked on one interface and silent on
// another is a gap in that file. Here the question is whether the *vocabulary*
// has evidence at all, so one fixture asserting it is enough.
type Evidence struct {
	// Axis is `detector`, `mixin` or `contract`, and Name the classification.
	Axis, Name string

	// Checked reports that some fixture's suite tier asserts it, and Modeled
	// that some fixture's model tier binds a law for it against a reference
	// the run can actually build.
	//
	// Both, rather than one `covered` bool, because the two are different
	// kinds of evidence and a reader triaging a red row needs to know which
	// one is missing. A classification with laws and no check may be complete;
	// one with neither is the finding.
	Checked, Modeled bool

	// Where names a fixture that supplied the evidence, empty where none did.
	// The corpus address a green row is answerable from.
	Where string
}

// Evidenced reports whether anything at all asserts this classification.
func (e Evidence) Evidenced() bool { return e.Checked || e.Modeled }

// Evidenced runs the real generators over the corpus in memory and reports, per
// registered classification, which tier asserts it.
//
// The same pipeline [Emission] runs and for the same reason: a subset would
// measure a production never runs. This walk reads the suite generator's own
// per-interface coverage rather than re-deriving it, because two derivations of
// "is this checked" are two chances to disagree — and the one that matters is
// the generator's, since that is what the consumer's header prints.
//
// Registered-but-unstamped classifications appear with neither flag set, which
// is correct: [Compare] is what reports them as a corpus gap, and a
// classification nothing stamps is also a classification nothing asserts.
func Evidenced(ctx context.Context, root string, patterns ...string) ([]Evidence, error) {
	pipe, err := runCorpus(ctx, root, patterns, corpusGenerators())
	if err != nil {
		return nil, err
	}

	checked, modeled := map[string]string{}, map[string]string{}
	for origin, c := range sdk.PendingByOrigin[*suite.Contract](pipe.Store().Emit()) {
		where := c.IfaceName
		if iface, ok := origin.(*sdk.Interface); ok {
			where = iface.Package + "." + iface.Name
		}
		for _, cov := range c.Coverage {
			// Asserted rather than Checked, because they answer different
			// questions and only one of them is this census's. Checked is per
			// carrier — a stamp on both halves of a pair reads as unchecked
			// wherever the check states the claim from one of them, which is
			// right for the header and wrong here. Elsewhere is the model
			// tier's answer and is gated on the tier actually running: a law
			// in the catalogue for a fixture the model generator refuses is
			// evidence nowhere.
			if cov.Asserted() && checked[cov.Name] == "" {
				checked[cov.Name] = where
			}
			if cov.Elsewhere() && modeled[cov.Name] == "" {
				modeled[cov.Name] = where
			}
		}
	}

	var out []Evidence
	for axis, names := range Registered() {
		for _, name := range names {
			e := Evidence{
				Axis:    axis,
				Name:    name,
				Checked: checked[name] != "",
				Modeled: modeled[name] != "",
			}
			if e.Where = checked[name]; e.Where == "" {
				e.Where = modeled[name]
			}
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b Evidence) int {
		if a.Axis != b.Axis {
			return cmp.Compare(a.Axis, b.Axis)
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return out, nil
}

// Unevidenced returns the classifications no tier asserts and no row argues,
// each as `<axis>/<name>`.
//
// The register's other direction is [UnevidencedClassifications] itself: a row
// for a classification some tier now asserts is a stale excuse, and
// [TestEveryUnevidencedRowIsStillTrue] deletes it by failing.
func Unevidenced(all []Evidence) []string {
	var out []string
	for _, e := range all {
		if e.Evidenced() {
			continue
		}
		if _, argued := UnevidencedClassifications[e.Name]; argued {
			continue
		}
		out = append(out, e.Axis+"/"+e.Name)
	}
	slices.Sort(out)
	return out
}

// ArguedButEvidenced returns the register rows a tier has since covered.
func ArguedButEvidenced(all []Evidence) []string {
	byName := make(map[string]Evidence, len(all))
	for _, e := range all {
		byName[e.Name] = e
	}
	var out []string
	for name := range maps.Keys(UnevidencedClassifications) {
		if byName[name].Evidenced() {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}
