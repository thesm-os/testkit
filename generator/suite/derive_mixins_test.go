// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_stamps_test.go is: these rules read
// mixin params through the unexported projection, which only
// [mixinParamsOf] over a stamped bag populates.
package suite

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"

	"go.thesmos.sh/testkit"
)

// The refusals, one case per branch.
//
// What a rule DERIVES is pinned by the corpus and by a proof beside
// every emitted row. What it REFUSES is not: a refusal is a claim the
// generated file deliberately does not make, so a branch that stopped
// firing would show up as silently absent coverage rather than as a
// failure — which is the one thing the corpus cannot catch, because a
// missing check looks exactly like a check that was never owed.
//
// Each case asserts the reason too, not merely that something was
// refused. The reason is what a consumer reads to decide whether to fix
// their interface or write the row by hand, and a rule refusing for the
// wrong stated cause sends them to the wrong place.

// withMixinParam stamps one mixin's KV argument on the method's own
// declaration and re-derives the projection from it.
//
// The declaration rather than a detached bag, which is the order the
// pipeline runs in — [sentinelReader] carries the account of what goes
// wrong the other way round.
func withMixinParam(m Method, mixin, param, value string) Method {
	shape.MixinParamKey(mixin, param).Set(m.Source.EnsureMeta(), value, "test")
	m.mixinParams = mixinParamsOf(m.Source.Meta(), m.Mixins)
	return m
}

// erroring gives the method an error result, which several rules
// require before they will state a claim about a refusal.
func erroring(m Method) Method {
	m.Returns = append(m.Returns, golang.Return{Local: "err", Error: true})
	return m
}

// answering gives the method a value result of the named type beside
// whatever it already returns.
func answering(m Method, typeName string) Method {
	m.Returns = append([]golang.Return{{
		Local:  "got",
		Source: storefixture.Named(typeName),
	}}, m.Returns...)
	return m
}

// takingFunc gives the method a func-typed parameter, which is what a
// hooks registrar needs before a callback can be installed through it.
func takingFunc(m Method) Method {
	m.Params = append(m.Params, golang.Param{
		Name:   "fn",
		Source: storefixture.Func(nil, nil),
	})
	m.ArgFields = append(m.ArgFields, "Fn")
	return m
}

// takingInt gives the method an integer parameter — a position, once
// something declares it one.
func takingInt(m Method) Method {
	m.Params = append(m.Params, golang.Param{
		Name:   "i",
		Source: storefixture.Named("int"),
	})
	m.ArgFields = append(m.ArgFields, "I")
	return m
}

// mixinRefusalCase is one rule, one shape it cannot state a claim
// about, and the substring its reason must carry.
type mixinRefusalCase struct {
	name  string
	rule  stampRule
	iface Iface
	m     Method
	why   string
}

func (c mixinRefusalCase) Name() string { return c.name }

func TestMixinRulesRefuseByName(t *testing.T) {
	t.Parallel()

	// The partner shapes the cases point at, built once.
	observer := erroring(answering(bareMethod("Total", ""), "int"))
	predicate := erroring(stampMethod("Validate", ""))
	sizer := erroring(answering(bareMethod("Len", ""), "int"))
	notAnInt := erroring(answering(bareMethod("Name", ""), "string"))

	testkit.TableTest(t, []mixinRefusalCase{
		{
			"sideeffect with no observer names no partner",
			sideEffectRule,
			stampIface(erroring(stampMethod("Touch", "", MixinSideEffect))),
			erroring(stampMethod("Touch", "", MixinSideEffect)),
			"names no partner",
		},
		{
			"sideeffect naming a stranger says so",
			sideEffectRule,
			stampIface(observer),
			withMixinParam(
				erroring(stampMethod("Touch", "", MixinSideEffect)),
				MixinSideEffect, MixinSideEffectParam, "Ghost",
			),
			"is not a method of this interface",
		},
		{
			"partition without an axis says two writes would land on two keys",
			partitionRule,
			stampIface(observer),
			withMixinParam(
				erroring(stampMethod("Put", "", MixinPartition)),
				MixinPartition, MixinPartitionRead, "Read",
			),
			"no axis names the parameter",
		},
		{
			"partition without a reader says nothing observes the boundary",
			partitionRule,
			stampIface(observer),
			withMixinParam(
				erroring(stampMethod("Put", "", MixinPartition)),
				MixinPartition, MixinPartitionAxis, "key",
			),
			"no read partner names",
		},
		{
			"hooks with no registrar names none",
			hooksRule,
			stampIface(observer),
			erroring(stampMethod("Fire", "", MixinHooks)),
			"names no registrar",
		},
		{
			"hooks naming a registrar that takes no function says so",
			hooksRule,
			stampIface(observer),
			withMixinParam(
				erroring(stampMethod("Fire", "", MixinHooks)),
				MixinHooks, MixinHooksParam, "Total",
			),
			"takes no function",
		},
		{
			"indexed with no sizer says any index would be a guess",
			indexedRule,
			stampIface(sizer),
			erroring(takingInt(stampMethod("At", "", MixinIndexed))),
			"no sizer names",
		},
		{
			"indexed whose sizer answers no integer says so",
			indexedRule,
			stampIface(notAnInt),
			withMixinParam(
				erroring(takingInt(stampMethod("At", "", MixinIndexed))),
				MixinIndexed, MixinIndexedBy, "Name",
			),
			"answers no integer",
		},
		{
			"indexed on a method taking no integer has no position to bound",
			indexedRule,
			stampIface(sizer),
			withMixinParam(
				erroring(stampMethod("At", "", MixinIndexed)),
				MixinIndexed, MixinIndexedBy, "Len",
			),
			"takes no integer argument",
		},
		{
			"nilsafe with no error channel cannot report the nil",
			nilSafeRule,
			stampIface(observer),
			stampMethod("Store", "", MixinNilSafe),
			"no error channel",
		},
		{
			"nilsafe with no nilable argument has no nil to hand it",
			nilSafeRule,
			stampIface(observer),
			erroring(stampMethod("Store", "", MixinNilSafe)),
			"no argument can hold nil",
		},
		{
			"orderafter with no predecessor says nothing names what it is early for",
			orderAfterRule,
			stampIface(observer),
			erroring(bareMethod("Commit", "", MixinOrderAfter)),
			"no predecessor names",
		},
		{
			"orderafter with no unready sentinel cannot tell refused from broken",
			orderAfterRule,
			stampIface(erroring(bareMethod("Prepare", ""))),
			withMixinParam(
				erroring(bareMethod("Commit", "", MixinOrderAfter)),
				MixinOrderAfter, MixinOrderAfterParam, "Prepare",
			),
			"no unready sentinel",
		},
		{
			"validates with no validator names none",
			validatesRule,
			stampIface(predicate),
			erroring(stampMethod("Store", "", MixinValidates)),
			"names no validator",
		},
		{
			"validates naming a stranger says so",
			validatesRule,
			stampIface(predicate),
			withMixinParam(
				erroring(stampMethod("Store", "", MixinValidates)),
				MixinValidates, MixinValidatesParam, "Ghost",
			),
			"is not a method of this interface",
		},
		{
			"validates whose validator gives no verdict says so",
			validatesRule,
			stampIface(stampMethod("Validate", "")),
			withMixinParam(
				erroring(stampMethod("Store", "", MixinValidates)),
				MixinValidates, MixinValidatesParam, "Validate",
			),
			"no error channel to give one through",
		},
	}, func(t *testing.T, tc mixinRefusalCase) {
		plans, refusals := tc.rule(tc.iface, tc.m, callOf(tc.m))
		testkit.Len(t, plans, 0, "a rule that cannot state its claim licenses nothing")
		testkit.Len(t, refusals, 1, "and names exactly one gap")
		testkit.Contains(t, refusals[0].Why, tc.why,
			"the reason sends a reader to the thing they have to change")
		testkit.Equal(t, refusals[0].Deriver, DeriverStamps, "the gap is attributed to its deriver")
		testkit.NotEqual(t, refusals[0].Remedy, "", "a named gap carries what closes it")
	})
}

// The positive path for the one rule whose partner has to be a
// particular SHAPE rather than merely present: a registrar is only a
// registrar if something can be registered through it.
func TestHooksRuleStatesItsClaimThroughAFuncTakingRegistrar(t *testing.T) {
	t.Parallel()

	register := takingFunc(bareMethod("OnEvent", ""))
	fire := withMixinParam(
		erroring(stampMethod("Fire", "", MixinHooks)),
		MixinHooks, MixinHooksParam, "OnEvent",
	)

	plans, refusals := hooksRule(stampIface(register, fire), fire, callOf(fire))
	testkit.Len(t, refusals, 0, "the registrar resolves and takes a function")
	testkit.Len(t, plans, 1, "so the claim is stated")
	testkit.Equal(t, plans[0].ID.Seg, "hooks", "under its own segment")
	testkit.Contains(t, plans[0].Claim, "OnEvent",
		"and the claim names where a hook is installed")
}

// The rules that refuse nothing, which is a decision rather than an
// omission: both state a claim about the SAME method called twice, so
// there is no partner to resolve and no shape to be wrong.
func TestRepeatRulesAlwaysState(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		rule stampRule
		seg  string
	}{
		{"idempotent", idempotentRule, "idempotent"},
		{"accumulates", accumulatesRule, "accumulates"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := erroring(bareMethod("Close", ""))
			plans, refusals := tc.rule(stampIface(m), m, callOf(m))
			testkit.Len(t, refusals, 0, "nothing to resolve, so nothing to refuse")
			testkit.Len(t, plans, 1, "the claim is stated")
			testkit.Equal(t, plans[0].ID.Seg, tc.seg, "under its own segment")
		})
	}
}
