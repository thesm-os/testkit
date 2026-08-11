// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite"
	"go.thesmos.sh/testkit/generator/tiers"
)

// unstamped is a projection method whose source carries no classification
// parameters at all — the smallest thing a stamp read can miss on.
func unstamped() *suite.Method {
	return &suite.Method{Sig: &golang.Sig{Source: &node.Method{}}}
}

// TestLawFieldRefusals pins the arms no armed fixture reaches from outside:
// the field kinds nothing renders, and the roles nothing resolves. Each must
// answer a reason — the header line that keeps the miss visible — rather than
// an empty field a template would render as a nil closure.
func TestLawFieldRefusals(t *testing.T) {
	t.Parallel()

	b := &Bindings{}

	t.Run("a defaulted field is omitted without a reason", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(b, nil, tiers.Field{Name: "Limit", Kind: tiers.KindDefault}, nil, nil)
		testkit.True(t, field == nil && reason == "",
			"the law's Check owns the value; the binding says nothing")
	})

	t.Run("a trace handle is the runner's, omitted without a reason", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(b, nil, tiers.Field{Name: "Trace", Kind: tiers.KindTrace}, nil, nil)
		testkit.True(t, field == nil && reason == "",
			"the runner binds it on any TraceBinder; a generated value would race that")
	})

	t.Run("an optional supplied field is omitted, a required one refused", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(b, nil,
			tiers.Field{Name: "Disturb", Kind: tiers.KindSupplied, From: "disturb", Optional: true}, nil, nil)
		testkit.True(t, field == nil && reason == "",
			"zero is sound, says the manifest, so the option stays the consumer's")

		field, reason = lawFieldOf(b, nil,
			tiers.Field{Name: "HappensBefore", Kind: tiers.KindSupplied, From: "happens-before"}, nil, nil)
		testkit.True(t, field == nil, "a required supply fills nothing")
		testkit.Assert(t, reason).Contains("happens-before",
			"and the reason names the option that would")
	})

	t.Run("a constant without its stamp is refused", func(t *testing.T) {
		t.Parallel()
		// The manifest names a stamp key; a declaration that does not carry it
		// has no value to render, and the reason names the missing stamp.
		field, reason := lawFieldOf(b, nil, tiers.Field{
			Name: "Sentinel", Kind: tiers.KindConstant, From: "shape.mixin.nonesuch.sentinel",
		}, unstamped(), nil)
		testkit.True(t, field == nil, "a constant fills nothing without its stamp")
		testkit.Assert(t, reason).Contains("nonesuch",
			"and the reason names the stamp a reader would add")
	})

	t.Run("an unknown kind is refused, not zero-filled", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(b, nil, tiers.Field{Name: "X", Kind: tiers.FieldKind("nonesuch")}, nil, nil)
		testkit.True(t, field == nil, "nothing renders what nothing defines")
		testkit.Assert(t, reason).Contains("unknown", "and the reason says so")
	})

	t.Run("a key handle without a projection is refused", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(b, nil, tiers.Field{Name: "KeyOf", Kind: tiers.KindHandle}, nil, nil)
		testkit.True(t, field == nil, "no projection was derived")
		testkit.Assert(t, reason).Contains("key projection", "and the reason names it")
	})

	t.Run("the roles nothing resolves are refused", func(t *testing.T) {
		t.Parallel()
		_, reason := roleMethod(b, nil, "family.reader", nil, nil)
		testkit.Assert(t, reason).Contains("no keyed reader",
			"the reader family needs a reader")
		_, reason = roleMethod(b, nil, "family.aggregator", nil, nil)
		testkit.Assert(t, reason).Contains("nothing resolves",
			"a family this build does not resolve says so rather than guessing")
	})
}
