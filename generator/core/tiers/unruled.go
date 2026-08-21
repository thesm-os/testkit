// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import (
	"slices"

	"go.thesmos.sh/testkit/core/lawid"
)

// UnruledLaws names every law [engine/model/law] ships that no rule in
// this catalogue can select, with why nothing selects it.
//
// A law with no rule is invisible: the engine implements it, the census
// counts it, and no generated file ever binds it, so it contributes
// nothing to any consumer's evidence and nothing reports the shortfall.
// That is the same silence [UnevidencedClassifications] exists for, one
// registry over — and it has the same fix, which is to make the absence
// something a build can read.
//
// A row here is an argument, not an exemption. Two of them are the
// argument that no rule COULD select the law as the vocabulary stands;
// a law that is merely unruled yet belongs in the catalogue, not here.
//
//nolint:gochecknoglobals // a register, read-only after init.
var UnruledLaws = map[string]string{
	lawid.PublisherDelivery: "the default arm of PublisherDeliveryBound's per-mode ID switch. " +
		"Each of the three modes has a rule and reports its own identifier, and a publisher " +
		"declaring no mode binds PublisherDelivers instead — so the arm is reachable only by " +
		"constructing the law by hand, which the generator never does",

	lawid.HashChainIntegrityErr: "the variant reading integrity off an Err() accessor rather " +
		"than a Verify() method. The chain contract names a verify role and nothing names an " +
		"integrity accessor, so no directive selects between the two. Blocked upstream on the " +
		"vocabulary, like retrysucceeds and scope: inventing a parameter here to make our own " +
		"selection land would move the problem into the consumer's annotation",
}

// Ruled reports every law some rule can select, sorted.
//
// The set [UnruledLaws] is the complement of, within what [lawid.All]
// declares. Both directions matter to the gate: a law in neither is one
// nothing accounts for, and a law in both is a row that outlived the
// gap it described.
func Ruled() []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		if !seen[r.Law] {
			seen[r.Law] = true
			out = append(out, r.Law)
		}
	}
	slices.Sort(out)
	return out
}

// Unaccounted returns the laws that neither a rule selects nor
// [UnruledLaws] argues for, sorted — the gap itself.
func Unaccounted() []string {
	ruled := Ruled()
	var out []string
	for _, l := range lawid.All() {
		if slices.Contains(ruled, l) {
			continue
		}
		if _, argued := UnruledLaws[l]; argued {
			continue
		}
		out = append(out, l)
	}
	slices.Sort(out)
	return out
}

// StaleUnruledRows returns the rows naming a law some rule does select,
// sorted — an argument that has outlived what it argued about.
//
// The other direction of the gate, and the one a register loses without
// noticing: a rule added for a law leaves its row behind, the row goes
// on asserting that nothing selects it, and the next reader believes the
// row.
func StaleUnruledRows() []string {
	ruled := Ruled()
	var out []string
	for l := range UnruledLaws {
		if slices.Contains(ruled, l) {
			out = append(out, l)
		}
	}
	slices.Sort(out)
	return out
}
