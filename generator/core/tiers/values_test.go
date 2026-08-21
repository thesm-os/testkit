// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers_test

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/core/tiers"
)

// TestValueRestrictionsNameLiveMixins holds the table to the registry: an
// entry whose spelling no mixin carries is a restriction nothing triggers,
// and the pool it was meant to narrow silently goes wide.
func TestValueRestrictionsNameLiveMixins(t *testing.T) {
	t.Parallel()

	live := map[string]bool{}
	for _, m := range mixins.All() {
		live[m.Name] = true
	}
	for _, name := range tiers.ValueRestrictions() {
		testkit.True(t, live[name], name+" is a mixin the registry declares")
		reason, restricted := tiers.ValueRestriction(name)
		testkit.True(t, restricted, name+" answers its own listing")
		testkit.True(t, reason != "", name+" carries a reason a header can print")
	}
}

// TestValueRestrictionVerdicts pins the two answers: validates restricts —
// the entry the whole table exists for — and a mixin with no claim on the
// value domain does not.
func TestValueRestrictionVerdicts(t *testing.T) {
	t.Parallel()

	reason, restricted := tiers.ValueRestriction("validates")
	testkit.True(t, restricted, "a validating subject may refuse a drawn value")
	testkit.Assert(t, reason).Contains("refus", "and the reason says so")

	_, restricted = tiers.ValueRestriction("causal")
	testkit.False(t, restricted, "an ordering claim says nothing about values")
}
