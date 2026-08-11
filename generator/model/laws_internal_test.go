// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/tiers"
)

// TestLawFieldRefusals pins the arms no armed fixture reaches from outside:
// the field kinds nothing renders, and the roles nothing resolves. Each must
// answer a reason — the header line that keeps the miss visible — rather than
// an empty field a template would render as a nil closure.
func TestLawFieldRefusals(t *testing.T) {
	t.Parallel()

	b := &Bindings{}

	t.Run("a defaulted field is omitted without a reason", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(b, tiers.Field{Name: "Limit", Kind: tiers.KindDefault}, nil, nil)
		testkit.True(t, field == nil && reason == "",
			"the law's Check owns the value; the binding says nothing")
	})

	t.Run("the kinds nothing renders name themselves", func(t *testing.T) {
		t.Parallel()
		for _, k := range []tiers.FieldKind{tiers.KindConstant, tiers.KindTrace, tiers.KindSupplied} {
			field, reason := lawFieldOf(b, tiers.Field{Name: "X", Kind: k}, nil, nil)
			testkit.True(t, field == nil, string(k)+" fills nothing")
			testkit.Assert(t, reason).Contains(string(k),
				"and the reason names the kind a reader acts on")
		}
	})

	t.Run("an unknown kind is refused, not zero-filled", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(b, tiers.Field{Name: "X", Kind: tiers.FieldKind("nonesuch")}, nil, nil)
		testkit.True(t, field == nil, "nothing renders what nothing defines")
		testkit.Assert(t, reason).Contains("unknown", "and the reason says so")
	})

	t.Run("a key handle without a projection is refused", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(b, tiers.Field{Name: "KeyOf", Kind: tiers.KindHandle}, nil, nil)
		testkit.True(t, field == nil, "no projection was derived")
		testkit.Assert(t, reason).Contains("key projection", "and the reason names it")
	})

	t.Run("the roles nothing resolves are refused", func(t *testing.T) {
		t.Parallel()
		_, reason := roleMethod("family.reader", nil, nil)
		testkit.Assert(t, reason).Contains("no keyed reader",
			"the reader family needs a reader")
		_, reason = roleMethod("family.aggregator", nil, nil)
		testkit.Assert(t, reason).Contains("nothing resolves",
			"a family this build does not resolve says so rather than guessing")
	})
}
