// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"slices"
	"strings"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/tiers"
)

// Coverage is one classification an interface carries, and what asserts it.
//
// Rendered into the harness header a reader meets before the checks, so an
// absent check is a stated boundary rather than a suspected defect. Without it
// a `deleteremoves` harness reads as thirteen checks with its declared mixin
// invisible, and a reader cannot tell "asserted somewhere else" from "silently
// ignored" — which is the tier-boundary form of what docs/adr/0017 exists to
// prevent.
//
// # Why the two halves are derived, not declared
//
// This used to carry a hand-written tier assigning each classification to one
// half of testkit's evidence. That encoded a coincidence as a rule: the suite
// tier checks fifteen classifications and none of them happens to have a law,
// so the two sets looked disjoint. They are not disjoint by nature — a
// classification can carry a property a fixed call sequence can state *and* a
// property only a reference comparison can. The table also drifted silently,
// naming one law per classification while the catalogue grew several for some
// and none for others.
//
// So both halves are computed. [Coverage.Checked] is this generator's own
// answer, and it is the only party that knows it. [Coverage.Laws] comes from
// [tiers.LawsFor], which is the same function the model bindings select
// through — so the header cannot claim a law the bindings do not register.
type Coverage struct {
	// Axis is `detector`, `mixin` or `contract`, and Name the classification.
	Axis, Name string

	// Methods names the methods carrying it, in method-set order.
	Methods []string

	// Laws are the identifiers the model bindings assert for it, empty where
	// no rule selects one.
	Laws []string

	// Checked reports whether this generator emitted a check for it here.
	//
	// Not the same as being assertable: `deprecated` generates nothing,
	// because it is a fact about a method rather than a claim about its
	// behaviour.
	Checked bool
}

// Elsewhere reports whether the evidence for this classification comes from
// somewhere other than a check a consumer could write.
//
// The distinction the header turns on. A law compares against a reference
// implementation a suite run has no way to build, so telling a consumer to
// hand-write one is bad advice; a classification nothing asserts is a gap, and
// the extension point is exactly where it belongs. Conflating the two produced
// a header that told consumers to write `deleteremoves` themselves, which
// nobody can state against a single subject.
func (c Coverage) Elsewhere() bool { return len(c.Laws) > 0 }

// checkedBy names the emit kinds that assert each classification.
//
// A mixin or a contract reports under its own name, so the subtest answers
// directly. A detector does not: `reader`'s check reports as "an error carries
// the zero value", which is what it asserts rather than what stamped it — so
// the pairing has to be written down, or a harness that checks a shape lists it
// as unchecked.
//
//nolint:gochecknoglobals // a lookup table, read-only after init
var checkedBy = map[string][]sdk.Kind{
	"reader": {KindZeroOnError},
	// Both members of the miss family, because which one a shape earns is
	// decided by the signature rather than by the stamp: a `readerwithbool`
	// whose bool the source dropped reports its miss in the value, and keying
	// on the flag form alone would list a checked shape as unchecked.
	"readernoerror":  {KindMissZero, KindMissFlag},
	"pointerreader":  {KindMissZero, KindMissFlag},
	"readerwithbool": {KindMissZero, KindMissFlag},
	"lookup":         {KindMissZero, KindMissFlag},
	"batchreader":    {KindBatchSize},
}

// DetectorCheck reports whether a kind is the check a shape stamp earns, rather
// than one the signature or a directive does.
//
// The three families name their subtests differently: a mixin's is the
// classification, and a detector's is what it asserts. So only the second needs
// the mapping, and only the second can be missing from it.
func DetectorCheck(kind sdk.Kind) bool {
	switch kind {
	case KindMissZero, KindMissFlag, KindBatchSize, KindZeroOnError:
		return true
	default:
		return false
	}
}

// ShapesCheckedBy returns the classifications a kind is the check for.
func ShapesCheckedBy(kind sdk.Kind) []string {
	var out []string
	for name, kinds := range checkedBy {
		if slices.Contains(kinds, kind) {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

// checksFor reports whether the method carries a check about this
// classification.
func checksFor(m Method, name string) bool {
	for _, ck := range m.Checks {
		if ck.Subtest == name {
			return true
		}
		if slices.Contains(checkedBy[name], ck.KindName) {
			return true
		}
	}
	return false
}

// coverageOf collects every classification the interface's methods carry.
//
// Union across the method set rather than per method, because the header is
// about the interface: a reader asking "what does this file not check" wants
// one list, and which methods carry each is what the entry names.
//
// The laws are read per classification rather than per method, which is the
// weaker of the two questions [tiers] answers and the right one here: the
// header says what covers a declaration, and a binding says what one method
// earns. Asking the stronger question would make the header depend on which
// method happened to be first.
func coverageOf(methods []Method) []Coverage {
	byName := map[string]*Coverage{}
	order := []string{}

	add := func(axis, name, method string, checked bool) {
		if name == "" {
			return
		}
		c, seen := byName[name]
		if !seen {
			c = &Coverage{Axis: axis, Name: name, Laws: tiers.LawsFor(name)}
			byName[name] = c
			order = append(order, name)
		}
		if !slices.Contains(c.Methods, method) {
			c.Methods = append(c.Methods, method)
		}
		c.Checked = c.Checked || checked
	}

	for _, m := range methods {
		add("detector", m.Shape(), m.Name, checksFor(m, m.Shape()))
		for _, name := range m.Mixins {
			add("mixin", name, m.Name, checksFor(m, name))
		}
		for _, name := range m.Contracts {
			add("contract", name, m.Name, checksFor(m, name))
		}
	}

	out := make([]Coverage, 0, len(order))
	for _, name := range order {
		out = append(out, *byName[name])
	}
	return out
}

// Shape returns the detector the annotator stamped on this method, empty when
// it stamped none.
func (m Method) Shape() string {
	if m.Source == nil {
		return ""
	}
	return shape.Get(m.Source.Meta())
}

// MethodList spells the methods carrying a classification, for the header.
func (c Coverage) MethodList() string { return strings.Join(c.Methods, ", ") }
