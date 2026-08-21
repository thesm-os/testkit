// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite"
)

// TestSuppliedDoors pins the door builder's arms: each shape spells its
// closure at the fixture's types or refuses with what is missing, a field
// shared by several laws builds one door, and a name asked at two shapes is
// a conflict rather than a shadow.
func TestSuppliedDoors(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	door := func(b *Bindings, law, field, from string, m *subject.Method) (*LawField, string) {
		r := tiers.Rule{Law: law, Fields: []tiers.Field{
			{Name: field, Kind: tiers.KindSupplied, From: from},
		}}
		return lawFieldOf(b, nil, r, r.Fields[0], m, nil)
	}

	t.Run("a key-typed door needs the keys pool", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		_, reason := door(b, lawid.CausalOrdering, "HappensBefore", "happens-before", nil)
		testkit.Assert(t, reason).Contains("key no method", "ClientOp is keyed")

		b.Keys = Pool{Type: sdk.Builtin(qStr)}
		field, reason := door(b, lawid.CausalOrdering, "HappensBefore", "happens-before", nil)
		testkit.True(t, reason == "" && field.Pool == "happensBefore",
			"the door opens at the pool's key: "+reason)
	})

	t.Run("an element-typed door reads the drained slice", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		scalar := projected("Items", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef(qStr)), errRet})
		_, reason := door(b, lawid.StreamStableOrder, "Less", "order", scalar)
		testkit.True(t, reason != "", "a non-slice drain spells no element")

		drain := projected("Items", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(sliceRef(namedRef(qStr))), errRet})
		field, reason := door(b, lawid.StreamStableOrder, "Less", "order", drain)
		testkit.True(t, reason == "" && field.Pool == "less", "the door opens at the element: "+reason)

		_, reason = door(b, lawid.StreamOverMatch, "Required", "required", drain)
		testkit.True(t, reason == "", "the list door shares the element: "+reason)
	})

	t.Run("the history door is one door for three laws", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}, Keys: Pool{Type: sdk.Builtin(qStr)}}
		_, reason := door(b, lawid.SnapshotIsolationG0, "History", "history", nil)
		testkit.True(t, reason == "", "the first isolation level opens the door: "+reason)
		_, reason = door(b, lawid.SnapshotIsolationG1, "History", "history", nil)
		testkit.True(t, reason == "", "the second reads the same one: "+reason)
		testkit.Equal(t, len(b.SuppliedOptions), 1, "one door, three laws")
	})

	t.Run("a name asked at two shapes is a conflict", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		testkit.Equal(t, b.addSuppliedOption(&SuppliedOption{Config: "x", Shape: supSubjPred}), "",
			"the first spelling lands")
		testkit.Assert(t, b.addSuppliedOption(&SuppliedOption{Config: "x", Shape: supStats})).
			Contains("second shape", "and the second is a conflict, not a shadow")
	})

	t.Run("the subject-only doors open unconditionally", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		_, reason := door(b, lawid.PoolLeakFree, "Balanced", "balanced", nil)
		testkit.True(t, reason == "", "the balance door: "+reason)
		_, reason = door(b, lawid.PoolBalanced, "Stats", "stats", nil)
		testkit.True(t, reason == "", "the stats door: "+reason)
		free := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}, Keys: Pool{Type: sdk.Builtin(qStr)}}
		_, reason = door(free, lawid.LeaseReleasedOnCancel, "Free", "free", nil)
		testkit.True(t, reason == "", "the free door: "+reason)
	})

	t.Run("the merge door reads the observation", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		_, reason := door(b, lawid.EventualConvergence, "Merge", "merge", nil)
		testkit.Assert(t, reason).Contains("observes state through no method",
			"no observation, no lattice to join")

		agg := projected("Count", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), res(namedRef("error"))})
		stamp(agg, "aggregator", "", "")
		h := &suite.Contract{Methods: []subject.Method{*agg}}
		field, reason := lawFieldOf(b, h, tiers.Rule{
			Law:    lawid.EventualConvergence,
			Fields: []tiers.Field{{Name: "Merge", Kind: tiers.KindSupplied, From: "merge"}},
		},
			tiers.Field{Name: "Merge", Kind: tiers.KindSupplied, From: "merge"}, nil, nil)
		testkit.True(t, reason == "" && field.Pool == "merge",
			"an aggregate is the lattice's state: "+reason)
	})

	t.Run("the replay doors open beside the drained log", func(t *testing.T) {
		t.Parallel()
		errRet := res(namedRef("error"))
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		replay := projected("Replay",
			[]golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(sliceRef(pkgRef("example.com/c", "Entry"))), errRet})
		carrier := projected("Append",
			[]golang.Param{arg("ctx", ctxRef()), arg("e", pkgRef("example.com/c", "Entry"))},
			[]golang.Return{errRet})
		shape.ContractPartnerKey("chain", "replay").Set(carrier.Source.EnsureMeta(), "Replay", "test")
		h := &suite.Contract{Methods: []subject.Method{*replay, *carrier}}
		field, reason := lawFieldOf(b, h, tiers.Rule{
			Law:    lawid.ReplayCausalOrdering,
			Fields: []tiers.Field{{Name: fEntryID, Kind: tiers.KindSupplied, From: "entry-id"}},
		},
			tiers.Field{Name: fEntryID, Kind: tiers.KindSupplied, From: "entry-id"}, carrier, nil)
		testkit.True(t, reason == "" && field.Pool == "entryID",
			"the entry door opens at the log's element: "+reason)
	})

	t.Run("the replay doors need the replay role", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		_, reason := door(b, lawid.ReplayCausalOrdering, fEntryID, "entry-id", unstamped())
		testkit.True(t, reason != "", "no chain.replay stamp, no entry to identify")
	})

	t.Run("a field the table does not transcribe keeps the refusal", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		_, reason := door(b, lawid.ReadAfterWrite, "Nonesuch", "nonesuch", nil)
		testkit.Assert(t, reason).Contains("no generated value can stand in for",
			"an untranscribed field is not a door")
	})
}

// TestWatcherMemberClosures pins the member-scope wiring: the keyed handle
// draw, the keyed write beside it, and the two closures derived from the
// next=/stop= member stamps — each binding at the fixture's types or
// refusing with what is missing.
func TestWatcherMemberClosures(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{
		Subject: subject.Subject{IfaceName: "Contract"},
		Keys:    Pool{Type: sdk.Builtin(qStr), Q: qStr},
		Values:  Pool{Type: sdk.Builtin("Value"), Q: "Value"},
	}
	watch := projected("Watch",
		[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
		[]golang.Return{res(namedRef("Subscription")), errRet})
	shape.ContractRoleKey("watcher").Set(watch.Source.EnsureMeta(), "watch", "test")
	shape.ContractPartnerKey("watcher", "trigger").Set(watch.Source.EnsureMeta(), "Trigger", "test")
	trigger := projected("Trigger",
		[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr)), arg("v", namedRef("Value"))},
		[]golang.Return{errRet})

	r := tiers.Rule{Law: lawid.WatcherReturnsOnChange, Fields: []tiers.Field{
		{Name: "Watch", Kind: tiers.KindRole, From: "watcher.watch"},
		{Name: "Mutate", Kind: tiers.KindRole, From: "watcher.trigger"},
		{Name: "Next", Kind: tiers.KindSupplied, From: memberNext},
		{Name: "Stop", Kind: tiers.KindSupplied, From: memberStop},
	}}
	h := harnessOf(watch, trigger)

	t.Run("the roles bind at their shapes", func(t *testing.T) {
		t.Parallel()
		field, reason := lawFieldOf(b, h, r, r.Fields[0], watch, nil)
		testkit.True(t, reason == "" && field.Out != nil, "the keyed handle binds: "+reason)

		field, reason = lawFieldOf(b, h, r, r.Fields[1], watch, nil)
		testkit.True(t, reason == "" && field.Value != nil, "the keyed write binds: "+reason)
	})

	t.Run("the member stamps derive their closures", func(t *testing.T) {
		t.Parallel()
		stamped := projected("Watch",
			[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
			[]golang.Return{res(namedRef("Subscription")), errRet})
		shape.ContractRoleKey("watcher").Set(stamped.Source.EnsureMeta(), "watch", "test")
		sdk.EnsureKey(paramWatcherKey+memberNext, sdk.StringParser).
			Set(stamped.Source.EnsureMeta(), "example.com/w.Subscription.Next", "test")
		sdk.EnsureKey(paramWatcherKey+memberStop, sdk.StringParser).
			Set(stamped.Source.EnsureMeta(), "example.com/w.Subscription.Stop", "test")

		field, reason := lawFieldOf(b, harnessOf(stamped, trigger), r, r.Fields[2], stamped, nil)
		testkit.True(t, reason == "", "the next member binds: "+reason)
		testkit.Equal(t, field.KeyField, "Next", "at the stamped member's local name")
		testkit.Equal(t, string(field.Kind()), "model.lawfield.MemberNext", "as the bounded read")

		field, reason = lawFieldOf(b, harnessOf(stamped, trigger), r, r.Fields[3], stamped, nil)
		testkit.True(t, reason == "", "the stop member binds: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.MemberStop", "as the teardown")
	})

	t.Run("an unstamped member refuses by name", func(t *testing.T) {
		t.Parallel()
		_, reason := lawFieldOf(b, h, r, r.Fields[2], watch, nil)
		testkit.Assert(t, reason).Contains("does not name", "no stamp, no member to call")
	})

	t.Run("a wide watch and a bare trigger refuse", func(t *testing.T) {
		t.Parallel()
		wide := projected("Watch",
			[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr)), arg("opts", namedRef(qStr))},
			[]golang.Return{res(namedRef("Subscription")), errRet})
		_, reason := bindField(b, lawid.WatcherReturnsOnChange, "Watch", wide)
		testkit.Assert(t, reason).Contains("does not watch one key", "two inputs watch nothing")

		bare := projected("Trigger",
			[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))}, []golang.Return{errRet})
		_, reason = bindField(b, lawid.WatcherReturnsOnChange, "Mutate", bare)
		testkit.Assert(t, reason).Contains("does not write one value under one key",
			"a valueless trigger publishes nothing the watch could compare")
	})
}

// TestMemberClosureResolutionClauses walks the member arm's remaining
// refusals: a rule that names no watch, a watch answering nothing, and a
// next member with no value pool to yield into.
func TestMemberClosureResolutionClauses(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}

	watch := projected("Watch",
		[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
		[]golang.Return{res(namedRef("Subscription")), errRet})
	shape.ContractRoleKey("watcher").Set(watch.Source.EnsureMeta(), "watch", "test")
	sdk.EnsureKey(paramWatcherKey+memberNext, sdk.StringParser).
		Set(watch.Source.EnsureMeta(), "example.com/w.Subscription.Next", "test")

	watchless := tiers.Rule{Law: lawid.WatcherReturnsOnChange, Fields: []tiers.Field{
		{Name: "Next", Kind: tiers.KindSupplied, From: memberNext},
	}}
	_, reason := lawFieldOf(b, harnessOf(watch), watchless, watchless.Fields[0], watch, nil)
	testkit.Assert(t, reason).Contains("does not name", "no watch field, no handle to read through")

	r := tiers.Rule{Law: lawid.WatcherReturnsOnChange, Fields: []tiers.Field{
		{Name: "Watch", Kind: tiers.KindRole, From: "watcher.watch"},
		{Name: "Next", Kind: tiers.KindSupplied, From: memberNext},
	}}
	flat := projected("Watch",
		[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))}, []golang.Return{errRet})
	shape.ContractRoleKey("watcher").Set(flat.Source.EnsureMeta(), "watch", "test")
	sdk.EnsureKey(paramWatcherKey+memberNext, sdk.StringParser).
		Set(flat.Source.EnsureMeta(), "example.com/w.Subscription.Next", "test")
	_, reason = lawFieldOf(b, harnessOf(flat), r, r.Fields[1], flat, nil)
	testkit.Assert(t, reason).Contains("answers none", "a flat watch has no handle")

	_, reason = lawFieldOf(b, harnessOf(watch), r, r.Fields[1], watch, nil)
	testkit.Assert(t, reason).Contains("yields a value no method here draws",
		"no pool, nothing for the read to answer")

	_, reason = bindField(b, lawid.WatcherReturnsOnChange, "Watch", flat)
	testkit.Assert(t, reason).Contains("answers no handle to read through",
		"the keyed handle refuses a flat watch too")
}

// TestMissSentinelAndDisturb pins the saturation fixes' derivations: the
// stamped miss sentinel routed into the oracle, and the adjacent-key
// disturbance derived from the driven writer — each omitted rather than
// guessed where its anchor is absent.
func TestMissSentinelAndDisturb(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))

	t.Run("a stamped sentinel routes and an unstamped one minted", func(t *testing.T) {
		t.Parallel()
		read := projected("Read",
			[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
			[]golang.Return{res(namedRef("Value")), errRet})
		read.Mixins = []string{"notfound"}
		shape.MixinParamKey("notfound", "sentinel").
			Set(read.Source.EnsureMeta(), "example.com/nf.ErrMissing", "test")
		sym := missSentinelOf(harnessOf(read))
		testkit.True(t, sym != nil, "the stamped sentinel is the oracle's miss")

		// A sentinel scoped to somebody else's condition is not this
		// one. deleteremoves names what a read AFTER A DELETE reports,
		// which coincides with a miss and is not one — the scan this
		// replaced took the first `sentinel=` it met on any mixin and
		// would have routed it as the oracle's miss.
		post := projected("Read",
			[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
			[]golang.Return{res(namedRef("Value")), errRet})
		post.Mixins = []string{"deleteremoves"}
		shape.MixinParamKey("deleteremoves", "sentinel").
			Set(post.Source.EnsureMeta(), "example.com/dr.ErrGone", "test")
		testkit.True(t, missSentinelOf(harnessOf(post)) == nil,
			"a post-delete sentinel is not the miss sentinel")

		bare := projected("Read",
			[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
			[]golang.Return{res(namedRef("Value")), errRet})
		testkit.True(t, missSentinelOf(harnessOf(bare)) == nil,
			"nothing stamped, nothing routed — the minted var stands")
	})

	t.Run("the disturbance derives from the feeding writer or stays omitted", func(t *testing.T) {
		t.Parallel()
		writer := stamp(projected("Store",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
			[]golang.Return{errRet}), "writer", "", "Value")
		b := &Bindings{
			Subject:   subject.Subject{IfaceName: "Mixed"},
			Values:    Pool{Type: sdk.Builtin("Value"), Q: "Value"},
			Keys:      Pool{Type: sdk.Builtin(qStr), Q: qStr, Field: "Key"},
			Actions:   []*Action{{Method: "Store", Pool: poolValues}, {Method: "Get", Pool: poolKeys}},
			Reference: Reference{KeyField: "Key"},
		}
		field := &LawField{BaseEmit: b.BaseEmit, Name: "Disturb", Iface: b.IfaceRef, Key: b.Keys.Type}
		got, reason := disturbFieldOf(b, harnessOf(writer), field, nil, nil)
		testkit.True(t, reason == "" && got != nil, "the writer-fed disturbance binds: "+reason)
		testkit.Equal(t, string(got.Kind()), "model.lawfield.DisturbWrite", "as the adjacent-key write")

		keyless := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		omitted, reason := disturbFieldOf(keyless, harnessOf(writer), field, nil, nil)
		testkit.True(t, omitted == nil && reason == "",
			"no pools, no projection — the field stays omitted, never guessed")
	})
}

// TestPublisherDrainDerivation pins the derived sweep's arms: it derives
// exactly where the subscribe role answers a channel, refuses everywhere
// else with what is missing, and derives once however many laws drain.
func TestPublisherDrainDerivation(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	drainRule := tiers.Rule{Law: lawid.PublisherDelivers, Fields: []tiers.Field{
		{Name: fDrain, Kind: tiers.KindSupplied, From: optDrain},
	}}
	chanReturn := func() golang.Return {
		ch := namedRef("chan")
		golang.MetaIsChannel.Set(ch.EnsureMeta(), true, "test")
		golang.MetaChanElem.Set(ch.EnsureMeta(), "example.com/p.Value", "test")
		return golang.Return{Type: sdk.Builtin("sub"), Source: ch}
	}
	subscribeWith := func(ret golang.Return) subject.Method {
		return *projectedReturns("Subscribe",
			[]golang.Param{arg("ctx", ctxRef())}, []golang.Return{ret, errRet})
	}
	carrier := func() *subject.Method {
		m := projected("Publish",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/p", "Value"))},
			[]golang.Return{errRet})
		shape.ContractPartnerKey("publisher", "subscribe").Set(m.Source.EnsureMeta(), "Subscribe", "test")
		return m
	}

	t.Run("a channel-answering subscribe derives the sweep once", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}
		h := &suite.Contract{Methods: []subject.Method{subscribeWith(chanReturn())}}
		m := carrier()
		field, reason := lawFieldOf(b, h, drainRule, drainRule.Fields[0], m, nil)
		testkit.True(t, reason == "" && field != nil, "the sweep derives: "+reason)
		testkit.Equal(t, field.KeyOfName, "drainSub", "through the property local the option outranks")
		testkit.True(t, b.Publisher != nil && b.Publisher.DrainName == "contractDrainSubscription",
			"and the file-level sweep is named once")

		again, reason := lawFieldOf(b, h, drainRule, drainRule.Fields[0], m, nil)
		testkit.True(t, reason == "" && again.KeyOfName == field.KeyOfName,
			"a second law reads the same derivation: "+reason)
	})

	t.Run("a subscription that answers no channel keeps the refusal", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}
		h := &suite.Contract{Methods: []subject.Method{subscribeWith(res(pkgRef("example.com/p", "Handle")))}}
		_, reason := lawFieldOf(b, h, drainRule, drainRule.Fields[0], carrier(), nil)
		testkit.Assert(t, reason).Contains("no channel", "an object handle is the drain option's territory")
	})

	t.Run("a carrier that stamps no subscribe partner refuses", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}
		h := &suite.Contract{Methods: []subject.Method{subscribeWith(chanReturn())}}
		unstampedCarrier := projected("Publish",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/p", "Value"))},
			[]golang.Return{res(namedRef("error"))})
		_, reason := lawFieldOf(b, h, drainRule, drainRule.Fields[0], unstampedCarrier, nil)
		testkit.Assert(t, reason).Contains("does not stamp", "the partner is the directive's to name")
	})
}

// TestPublisherPoolAndDrainRefusals pins the remaining arms: the messages
// pool rides the values pool where one exists, a law pool redeclared at a
// second type refuses, and a drain over the wrong subscription says which
// half is missing.
func TestPublisherPoolAndDrainRefusals(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))

	t.Run("messages ride the values pool", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject: subject.Subject{IfaceName: "Contract"},
			Values:  Pool{Type: sdk.Builtin(qStr), Q: qStr, Field: "Body"},
			Actions: []*Action{{Method: "Publish", Pool: "values"}},
		}
		r := tiers.Rule{Law: lawid.PublisherDelivers, Fields: []tiers.Field{
			{Name: "Messages", Kind: tiers.KindGenerator, From: "messages"},
		}}
		field, reason := lawFieldOf(b, nil, r, r.Fields[0], nil, nil)
		testkit.True(t, reason == "" && field.Pool == "values",
			"one pool, colliding by construction: "+reason)
	})

	t.Run("a law pool redeclared at a second type refuses", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}
		b.LawPools = append(b.LawPools, LawPool{Name: "payloads", Q: "int", Elem: sdk.Builtin("int")})
		r := tiers.Rule{Law: lawid.XSSSafe, Fields: []tiers.Field{
			{Name: "Payloads", Kind: tiers.KindGenerator, From: "payloads"},
		}}
		_, reason := lawFieldOf(b, nil, r, r.Fields[0], nil, nil)
		testkit.True(t, reason != "", "two laws asking one name at two types are caught")

		b2 := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}
		b2.LawPools = append(b2.LawPools, LawPool{Name: "offsets", Q: builtin64, Elem: sdk.Builtin(builtin64)})
		r2 := tiers.Rule{Law: lawid.ScheduledFiresAfterAdvance, Fields: []tiers.Field{
			{Name: "Offsets", Kind: tiers.KindGenerator, From: "offsets"},
		}}
		_, reason = lawFieldOf(b2, nil, r2, r2.Fields[0], nil, nil)
		testkit.True(t, reason != "", "the offsets pool holds one type too")
	})

	t.Run("a subscription answering nothing refuses the drain", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}
		bare := projectedReturns("Subscribe", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet})
		h := &suite.Contract{Methods: []subject.Method{*bare}}
		m := projected("Publish",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/p", "Value"))},
			[]golang.Return{errRet})
		shape.ContractPartnerKey("publisher", "subscribe").Set(m.Source.EnsureMeta(), "Subscribe", "test")
		r := tiers.Rule{Law: lawid.PublisherDelivers, Fields: []tiers.Field{
			{Name: fDrain, Kind: tiers.KindSupplied, From: optDrain},
		}}
		_, reason := lawFieldOf(b, h, r, r.Fields[0], m, nil)
		testkit.Assert(t, reason).Contains("nothing to observe", "no result, no channel to sweep")
	})

	t.Run("a channel whose element no stamp names refuses", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}
		ch := namedRef("chan")
		golang.MetaIsChannel.Set(ch.EnsureMeta(), true, "test")
		sub := projectedReturns("Subscribe", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{{Type: sdk.Builtin("sub"), Source: ch}, errRet})
		h := &suite.Contract{Methods: []subject.Method{*sub}}
		m := projected("Publish",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/p", "Value"))},
			[]golang.Return{errRet})
		shape.ContractPartnerKey("publisher", "subscribe").Set(m.Source.EnsureMeta(), "Subscribe", "test")
		r := tiers.Rule{Law: lawid.PublisherDelivers, Fields: []tiers.Field{
			{Name: fDrain, Kind: tiers.KindSupplied, From: optDrain},
		}}
		_, reason := lawFieldOf(b, h, r, r.Fields[0], m, nil)
		testkit.Assert(t, reason).Contains("no stamp names", "the sweep is typed at the element")
	})
}
