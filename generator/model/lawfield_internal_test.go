// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/suite"
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
		field, reason := lawFieldOf(
			b,
			nil,
			tiers.Rule{},
			tiers.Field{Name: "Limit", Kind: tiers.KindDefault},
			nil,
			nil,
		)
		testkit.True(t, field == nil && reason == "",
			"the law's Check owns the value; the binding says nothing")
	})

	t.Run("a trace handle is the runner's, omitted without a reason", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(
			b,
			nil,
			tiers.Rule{},
			tiers.Field{Name: "Trace", Kind: tiers.KindTrace},
			nil,
			nil,
		)
		testkit.True(t, field == nil && reason == "",
			"the runner binds it on any TraceBinder; a generated value would race that")
	})

	t.Run("an optional supplied field is omitted, a required one refused", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(
			b,
			nil,
			tiers.Rule{},
			tiers.Field{
				Name:     "Disturb",
				Kind:     tiers.KindSupplied,
				From:     "disturb",
				Optional: true,
			},
			nil,
			nil,
		)
		testkit.True(t, field == nil && reason == "",
			"zero is sound, says the manifest, so the option stays the consumer's")

		field, reason = lawFieldOf(
			b,
			nil,
			tiers.Rule{},
			tiers.Field{
				Name: "HappensBefore",
				Kind: tiers.KindSupplied,
				From: "happens-before",
			},
			nil,
			nil,
		)
		testkit.True(t, field == nil, "a required supply fills nothing")
		testkit.Assert(t, reason).Contains("happens-before",
			"and the reason names the option that would")
	})

	t.Run("a constant without its stamp is refused", func(t *testing.T) {
		t.Parallel()
		// The manifest names a stamp key; a declaration that does not carry it
		// has no value to render, and the reason names the missing stamp.
		field, reason := lawFieldOf(b, nil, tiers.Rule{}, tiers.Field{
			Name: "Sentinel", Kind: tiers.KindConstant, From: "shape.mixin.nonesuch.sentinel",
		}, unstamped(), nil)
		testkit.True(t, field == nil, "a constant fills nothing without its stamp")
		testkit.Assert(t, reason).Contains("nonesuch",
			"and the reason names the stamp a reader would add")
	})

	t.Run("an unknown kind is refused, not zero-filled", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(b, nil, tiers.Rule{},
			tiers.Field{Name: "X", Kind: tiers.FieldKind("nonesuch")}, nil, nil)
		testkit.True(t, field == nil, "nothing renders what nothing defines")
		testkit.Assert(t, reason).Contains("unknown", "and the reason says so")
	})

	t.Run("a key handle without a projection is refused", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(b, nil, tiers.Rule{},
			tiers.Field{Name: "KeyOf", Kind: tiers.KindHandle, From: "key-projection"}, nil, nil)
		testkit.True(t, field == nil, "no projection was derived")
		testkit.Assert(t, reason).Contains("key projection", "and the reason names it")
	})

	t.Run("the roles nothing resolves are refused", func(t *testing.T) {
		t.Parallel()
		_, reason := roleMethod(b, nil, "family.reader", nil, nil)
		testkit.Assert(t, reason).Contains("no keyed reader",
			"the reader family needs a reader")
		_, reason = roleMethod(b, nil, "family.aggregator", nil, nil)
		testkit.Assert(t, reason).Contains("no aggregate",
			"the aggregator family needs an aggregate")
		_, reason = roleMethod(b, nil, "family.nonesuch", nil, nil)
		testkit.Assert(t, reason).Contains("nothing resolves",
			"a family this build does not resolve says so rather than guessing")
	})
}

func TestResolveArgArms(t *testing.T) {
	t.Parallel()

	b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}

	t.Run("the pools refuse where nothing draws them", func(t *testing.T) {
		t.Parallel()
		_, reason := resolveArg(b, nil, tiers.Rule{}, tiers.BindKey, nil, nil)
		testkit.Assert(t, reason).Contains("key type", "no keyed method, no key argument")
		_, reason = resolveArg(b, nil, tiers.Rule{}, tiers.BindValue, nil, nil)
		testkit.Assert(t, reason).Contains("value type", "no valued method, no value argument")
	})

	t.Run("the partition is the anonymous single one", func(t *testing.T) {
		t.Parallel()
		ref, reason := resolveArg(b, nil, tiers.Rule{}, tiers.BindPartition, nil, nil)
		testkit.True(t, reason == "" && ref != nil, "one partition, spelled string")
	})

	t.Run("an observation argument reports the missing observation", func(t *testing.T) {
		t.Parallel()
		_, reason := resolveArg(b, harnessOf(), tiers.Rule{}, tiers.BindObservation, nil, nil)
		testkit.Assert(t, reason).Contains("observes state through no method",
			"nothing to observe is a named refusal")
	})

	t.Run("an unresolvable spelling refuses by name", func(t *testing.T) {
		t.Parallel()
		_, reason := resolveArg(b, nil, tiers.Rule{}, tiers.BindArg("bogus"), nil, nil)
		testkit.Assert(t, reason).Contains("nothing resolves", "an unknown argument names itself")
	})

	t.Run("the field-qualified forms read the role's own types", func(t *testing.T) {
		t.Parallel()
		errRet := res(namedRef("error"))
		m := projected("Classify",
			[]golang.Param{arg("ctx", ctxRef()), arg("in", namedRef(qStr))},
			[]golang.Return{res(namedRef(qStr)), errRet})
		r := roleRule(lawid.TotalOver, "Call")

		ref, reason := resolveArg(b, nil, r, tiers.InputOf("Call"), m, nil)
		testkit.True(t, reason == "" && ref != nil, "the input form reads the parameter: "+reason)
		ref, reason = resolveArg(b, nil, r, tiers.ResultOf("Call"), m, nil)
		testkit.True(t, reason == "" && ref != nil, "the result form reads the return: "+reason)
		ref, reason = resolveArg(b, nil, r, tiers.ScalarOf("Call"), m, nil)
		testkit.True(t, reason == "" && ref != nil, "the scalar form reads the return: "+reason)

		_, reason = resolveArg(b, nil, r, tiers.InputOf("Call"),
			projected("Count", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("int")), errRet}), nil)
		testkit.Assert(t, reason).Contains("takes none", "no input, no input argument")

		_, reason = resolveArg(b, nil, r, tiers.ElemOf("Call"), m, nil)
		testkit.Assert(t, reason).Contains("streams elements no stamp names",
			"a non-stream role yields no element")

		_, reason = resolveArg(b, nil, r, tiers.ResultOf("Bogus"), m, nil)
		testkit.Assert(t, reason).Contains("does not name",
			"a form naming an absent field refuses by name")

		nonRole := tiers.Rule{Fields: []tiers.Field{{Name: "Limit", Kind: tiers.KindDefault}}}
		_, reason = resolveArg(b, nil, nonRole, tiers.ResultOf("Limit"), m, nil)
		testkit.Assert(t, reason).Contains("not a role field",
			"a form naming a non-role field refuses by kind")
	})
}

func TestObservationOf(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	keyed := stamp(projected("Get", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
		[]golang.Return{res(namedRef(qStr)), errRet}), "reader", qStr, qStr)
	pooled := &Bindings{
		Subject: suite.Subject{IfaceName: "Mixed"},
		Keys:    Pool{Field: fieldKey, Q: qStr},
		Actions: []*Action{{Pool: poolKeys}},
	}

	t.Run("the drain outranks the aggregate outranks the keyed read", func(t *testing.T) {
		t.Parallel()
		collector := stamp(projected("Items", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(sliceRef(namedRef(qStr))), errRet}), "aggregator", "", qStr)
		agg := stamp(projected("Total", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), errRet}), "aggregator", "", "int")

		obs, reason := observationOf(pooled, harnessOf(collector, agg, keyed), nil)
		testkit.True(t, reason == "", "a drain observes everything: "+reason)
		testkit.Equal(t, obs.Method.Name, "Items", "so it wins")

		obs, reason = observationOf(pooled, harnessOf(agg, keyed), nil)
		testkit.True(t, reason == "", "an aggregate observes the whole: "+reason)
		testkit.Equal(t, obs.Method.Name, "Total", "so it beats the keyed read")

		obs, reason = observationOf(pooled, harnessOf(keyed), nil)
		testkit.True(t, reason == "" && obs.Keyed, "the fixture-keyed read is the floor: "+reason)
	})

	t.Run("the keyed fallback rides the projection's own reader", func(t *testing.T) {
		t.Parallel()
		obs, reason := observationOf(pooled, harnessOf(), keyed)
		testkit.True(t, reason == "" && obs.Keyed, "the resolved reader stands in: "+reason)
	})

	t.Run("nothing observable is a named refusal", func(t *testing.T) {
		t.Parallel()
		_, reason := observationOf(pooled, harnessOf(), nil)
		testkit.Assert(t, reason).Contains("no drain, no aggregate, no keyed read",
			"the refusal lists what would have served")
	})
}
