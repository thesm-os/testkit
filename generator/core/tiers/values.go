// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

// ValueRestriction reports whether the named mixin's claim narrows the set of
// values the subject accepts, and where it does, the reason fragment the
// generated header composes into its explanation.
//
// The model tier widens its value pools past the fixture pair wherever the
// contract licenses it: a writer with no restricting claim accepts any value
// of its type, so a subject refusing one is a finding rather than noise. A
// restricting claim inverts the license — the subject may refuse values no
// derivation can predict — and every pooled value must stay one the harness
// has proven accepted. The pool then keeps to the fixture pair, and the
// header names the claim so the consumer knows the values option is the door
// to a wider one.
//
// Data because the difference is semantic, here because this is the module
// the census closes: the table is held to the live mixin registry by test, so
// a spelling drifting upstream fails by name.
func ValueRestriction(mixin string) (string, bool) {
	reason, restricted := restrictedValues[mixin]
	return reason, restricted
}

// ValueRestrictions returns every restricted mixin's name, for the census.
func ValueRestrictions() []string {
	out := make([]string, 0, len(restrictedValues))
	for name := range restrictedValues {
		out = append(out, name)
	}
	return out
}

// restrictedValues names the mixins whose claim narrows the accepted value
// domain, each with the reason fragment the generated header prints. The
// fragment follows "the <mixin> claim on <method> ", so it opens with a verb.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var restrictedValues = map[string]string{
	mixinValidates: "licenses refusing values no derivation can predict",
	mixinSample:    "routes inputs through a builder a raw draw may fall outside of",
}
