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
)

func TestHandleFieldArms(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}, Values: Pool{Q: qStr}}

	handle := func(b *Bindings, law, name, from string, m *subject.Method, extra ...tiers.Field) (*LawField, string) {
		r := tiers.Rule{Law: law, Fields: append(extra, tiers.Field{
			Name: name, Kind: tiers.KindHandle, From: from,
		})}
		return lawFieldOf(b, nil, r, r.Fields[len(r.Fields)-1], m, nil)
	}

	t.Run("the constructed handles bind", func(t *testing.T) {
		t.Parallel()
		field, reason := handle(b, lawid.Associative, "Factory", "subject-factory", nil)
		testkit.True(t, reason == "" && field != nil, "the property's factory stands in: "+reason)

		field, reason = handle(b, lawid.AppendOnlyGrows, "Partitions", "partitions", nil)
		testkit.True(t, reason == "" && field != nil, "the anonymous partition set: "+reason)

		monotonic := projected("Version", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int64")), errRet})
		field, reason = handle(b, lawid.MonotonicNonDecreasing, "Less", "natural-order", monotonic)
		testkit.True(t, reason == "" && field.Out != nil, "a builtin scalar orders itself: "+reason)

		named := projected("State", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("example.com/x", "State")), errRet})
		_, reason = handle(b, lawid.MonotonicNonDecreasing, "Less", "natural-order", named)
		testkit.Assert(t, reason).Contains("the language does not", "no order on a named type")
	})

	t.Run("the identity hash follows the drain where one exists", func(t *testing.T) {
		t.Parallel()
		drainField := tiers.Field{Name: "Drain", Kind: tiers.KindRole, From: "self"}
		lister := projected("List", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(sliceRef(namedRef(qStr))), errRet})
		field, reason := handle(b, lawid.StreamNoDuplicates, "Hash", "identity-hash", lister, drainField)
		testkit.True(t, reason == "" && field.Value != nil, "the drained element hashes itself: "+reason)

		field, reason = handle(b, lawid.StreamNoDuplicates, "Hash", "identity-hash", nil)
		testkit.True(t, reason == "" && field != nil, "the values pool stands in without a drain: "+reason)
	})

	t.Run("the observation handle picks its spelling", func(t *testing.T) {
		t.Parallel()
		pooled := &Bindings{
			Subject: subject.Subject{IfaceName: "Mixed"},
			Keys:    Pool{Field: fieldKey, Q: qStr},
			Actions: []*Action{{Pool: poolKeys}},
		}
		keyed := stamp(projected("Get", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(namedRef(qStr)), errRet}), "reader", qStr, qStr)

		r := tiers.Rule{Law: lawid.IdempotentWrite, Fields: []tiers.Field{
			{Name: "Observe", Kind: tiers.KindHandle, From: "observation"},
		}}
		field, reason := lawFieldOf(pooled, harnessOf(keyed), r, r.Fields[0], nil, nil)
		testkit.True(t, reason == "", "the keyed observation binds: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.ObserveKeyed", "at the fixture key")

		agg := stamp(projected("Total", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), errRet}), "aggregator", "", "int")
		field, reason = lawFieldOf(pooled, harnessOf(agg), r, r.Fields[0], nil, nil)
		testkit.True(t, reason == "", "the aggregate observation binds: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.ObserveCall", "as a call")
	})

	t.Run("the handles nothing constructs refuse by name", func(t *testing.T) {
		t.Parallel()
		for from, needle := range map[string]string{
			"trace-classifier": "no keyed reader",
			// Both construct now; asked with nothing to resolve through, the
			// refusal says which half is missing.
			"history":        "no selecting method stamps",
			"coalesce-probe": "the manifest does not name",
		} {
			_, reason := handle(b, lawid.SingleflightCoalesces, "X", from, nil)
			testkit.Assert(t, reason).Contains(needle, from+" names its debt")
		}

		field, reason := handle(b, lawid.TTLExpiry, "Advance", "clock", nil)
		testkit.True(t, reason == "" && field != nil,
			"the clock handle binds — the template guards it on ModelClocked: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.Advance",
			"through the advance spelling")
	})
}

// TestCoalesceHandlesAndIdentityKey pins the two handle arms the family
// added: the probe/counter pair the coalescing law instruments itself with,
// and the key projection's identity fallback for the walk.
func TestCoalesceHandlesAndIdentityKey(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}
	run := projected("Run",
		[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr)), arg("compute", funcRef())},
		[]golang.Return{res(namedRef(qStr)), errRet})

	t.Run("the coalesce pair constructs beside its call role", func(t *testing.T) {
		t.Parallel()
		r := tiers.Rule{Law: lawid.SingleflightCoalesces, Fields: []tiers.Field{
			{Name: fCall, Kind: tiers.KindRole, From: fromSelf},
			{Name: "Compute", Kind: tiers.KindHandle, From: "coalesce-probe"},
			{Name: "Counter", Kind: tiers.KindHandle, From: "coalesce-counter"},
		}}
		field, reason := lawFieldOf(b, nil, r, r.Fields[1], run, nil)
		testkit.True(t, reason == "" && field.Out != nil, "the probe binds: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.Compute", "as the locked closure")

		field, reason = lawFieldOf(b, nil, r, r.Fields[2], run, nil)
		testkit.True(t, reason == "", "the counter binds: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.Counter", "by address")
	})

	t.Run("the key projection falls back to identity for the walk", func(t *testing.T) {
		t.Parallel()
		pageRead := projected("Page",
			[]golang.Param{arg("ctx", ctxRef()), arg("cur", namedRef("Cursor"))},
			[]golang.Return{res(sliceRef(namedRef("Value"))), res(namedRef("Cursor")), res(namedRef("bool")), errRet})
		r := tiers.Rule{Law: lawid.PaginatorNoDuplicates, Fields: []tiers.Field{
			{Name: "Page", Kind: tiers.KindRole, From: fromSelf},
			{Name: "KeyOf", Kind: tiers.KindHandle, From: "key-projection"},
		}}
		field, reason := lawFieldOf(b, nil, r, r.Fields[1], pageRead, nil)
		testkit.True(t, reason == "" && field.Value != nil, "identity stands in: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.KeyOfIdentity",
			"the element is its own fingerprint")

		// Any other law keeps the refusal: identity is the walk's answer, not
		// a general substitute for the projection.
		other := tiers.Rule{Law: lawid.UpdaterReplaces, Fields: []tiers.Field{
			{Name: "KeyOf", Kind: tiers.KindHandle, From: "key-projection"},
		}}
		_, reason = lawFieldOf(b, nil, other, other.Fields[0], pageRead, nil)
		testkit.Assert(t, reason).Contains("was not derivable", "no projection, no keyed law")
	})
}

// TestVersionStampAndHistoryHandles pins the small pair's two handles: the
// version-coherent draw over the cas cell, and the append-recording history
// the no-drops law reads.
func TestVersionStampAndHistoryHandles(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}

	t.Run("the version stamp reads the cell and names its member", func(t *testing.T) {
		t.Parallel()
		put := projected("Put",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))}, []golang.Return{errRet})
		shape.ContractRoleKey("cas").Set(put.Source.EnsureMeta(), "writer", "test")
		sdk.EnsureKey(paramCASVersion, sdk.StringParser).Set(put.Source.EnsureMeta(), "Version", "test")
		get := projected("Get", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("Value")), errRet})
		r := tiers.Rule{Law: lawid.CASAtomicOneWinner, Fields: []tiers.Field{
			{Name: "CAS", Kind: tiers.KindRole, From: "cas.writer"},
			{Name: fRead, Kind: tiers.KindRole, From: "family.cell"},
			{Name: "Stamp", Kind: tiers.KindHandle, From: handleVersionStamp},
		}}
		field, reason := lawFieldOf(b, harnessOf(put, get), r, r.Fields[2], put, nil)
		testkit.True(t, reason == "", "the stamp binds: "+reason)
		testkit.Equal(t, field.Method, "Get", "reading through the cell")
		testkit.Equal(t, field.KeyField, "Version", "at the named member")

		unstamped := projected("Put",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))}, []golang.Return{errRet})
		shape.ContractRoleKey("cas").Set(unstamped.Source.EnsureMeta(), "writer", "test")
		_, reason = lawFieldOf(b, harnessOf(unstamped, get), r, r.Fields[2], unstamped, nil)
		testkit.Assert(t, reason).Contains("names none", "no version member, no coherent draw")
	})

	t.Run("the history rides the append role beside the replay", func(t *testing.T) {
		t.Parallel()
		app := projected("Append",
			[]golang.Param{arg("ctx", ctxRef()), arg("e", namedRef("Entry"))}, []golang.Return{errRet})
		shape.ContractRoleKey("chain").Set(app.Source.EnsureMeta(), "append", "test")
		shape.ContractPartnerKey("chain", "replay").Set(app.Source.EnsureMeta(), "Replay", "test")
		replay := projected("Replay", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(sliceRef(namedRef("Entry"))), errRet})
		r := tiers.Rule{Law: lawid.AppendOnlyNoDrops, Fields: []tiers.Field{
			{Name: fReplay, Kind: tiers.KindRole, From: "chain.replay"},
			{Name: "History", Kind: tiers.KindHandle, From: handleHistoryLog},
		}}
		field, reason := lawFieldOf(b, harnessOf(app, replay), r, r.Fields[1], app, nil)
		testkit.True(t, reason == "" && field.Value != nil, "the history binds: "+reason)
		testkit.Equal(t, field.Method, "Append", "riding the append the inert check watches")
		testkit.Equal(t, string(field.Kind()), "model.lawfield.HistoryRef", "as the shared local")

		bare := tiers.Rule{Law: lawid.AppendOnlyNoDrops, Fields: []tiers.Field{
			{Name: "History", Kind: tiers.KindHandle, From: handleHistoryLog},
		}}
		_, reason = lawFieldOf(b, harnessOf(app, replay), bare, bare.Fields[0], app, nil)
		testkit.Assert(t, reason).Contains("does not name", "no replay field, no element to log")
	})
}

// TestVersionStampAndHistoryRefusals walks the pair's remaining clauses:
// each is a distinct wrong fixture whose refusal must name the missing half.
func TestVersionStampAndHistoryRefusals(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}

	get := projected("Get", []golang.Param{arg("ctx", ctxRef())},
		[]golang.Return{res(namedRef("Value")), errRet})

	t.Run("a stamp without its call or its attempt refuses", func(t *testing.T) {
		t.Parallel()
		put := projected("Put",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))}, []golang.Return{errRet})
		shape.ContractRoleKey("cas").Set(put.Source.EnsureMeta(), "writer", "test")
		sdk.EnsureKey(paramCASVersion, sdk.StringParser).Set(put.Source.EnsureMeta(), "Version", "test")

		bare := tiers.Rule{Law: lawid.CASAtomicOneWinner, Fields: []tiers.Field{
			{Name: fRead, Kind: tiers.KindRole, From: "family.cell"},
			{Name: "Stamp", Kind: tiers.KindHandle, From: handleVersionStamp},
		}}
		_, reason := lawFieldOf(b, harnessOf(put, get), bare, bare.Fields[1], put, nil)
		testkit.Assert(t, reason).Contains("does not name", "no CAS field, no attempt to stamp")

		nullary := projected("Put", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet})
		shape.ContractRoleKey("cas").Set(nullary.Source.EnsureMeta(), "writer", "test")
		sdk.EnsureKey(paramCASVersion, sdk.StringParser).Set(nullary.Source.EnsureMeta(), "Version", "test")
		r := tiers.Rule{Law: lawid.CASAtomicOneWinner, Fields: []tiers.Field{
			{Name: "CAS", Kind: tiers.KindRole, From: "cas.writer"},
			{Name: fRead, Kind: tiers.KindRole, From: "family.cell"},
			{Name: "Stamp", Kind: tiers.KindHandle, From: handleVersionStamp},
		}}
		_, reason = lawFieldOf(b, harnessOf(nullary, get), r, r.Fields[2], nullary, nil)
		testkit.Assert(t, reason).Contains("it takes none", "nothing to stamp a version into")
	})

	t.Run("a history over a streaming replay refuses", func(t *testing.T) {
		t.Parallel()
		app := projected("Append",
			[]golang.Param{arg("ctx", ctxRef()), arg("e", namedRef("Entry"))}, []golang.Return{errRet})
		shape.ContractRoleKey("chain").Set(app.Source.EnsureMeta(), "append", "test")
		shape.ContractPartnerKey("chain", "replay").Set(app.Source.EnsureMeta(), "Replay", "test")
		iterReplay := projected("Replay", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("Seq")), errRet})
		r := tiers.Rule{Law: lawid.AppendOnlyNoDrops, Fields: []tiers.Field{
			{Name: fReplay, Kind: tiers.KindRole, From: "chain.replay"},
			{Name: "History", Kind: tiers.KindHandle, From: handleHistoryLog},
		}}
		_, reason := lawFieldOf(b, harnessOf(app, iterReplay), r, r.Fields[1], app, nil)
		testkit.Assert(t, reason).Contains("no stamp names", "an unnamed stream logs nothing typed")
	})
}

// TestVersionStampCellAndReplayResolution pins the pair's resolution
// clauses: a stamp whose rule names no cell read, and a history whose
// replay partner names a method the interface does not have.
func TestVersionStampCellAndReplayResolution(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: subject.Subject{IfaceName: "Contract"}}

	put := projected("Put",
		[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))}, []golang.Return{errRet})
	shape.ContractRoleKey("cas").Set(put.Source.EnsureMeta(), "writer", "test")
	sdk.EnsureKey(paramCASVersion, sdk.StringParser).Set(put.Source.EnsureMeta(), "Version", "test")
	cellless := tiers.Rule{Law: lawid.CASAtomicOneWinner, Fields: []tiers.Field{
		{Name: fCAS, Kind: tiers.KindRole, From: "cas.writer"},
		{Name: "Stamp", Kind: tiers.KindHandle, From: handleVersionStamp},
	}}
	_, reason := lawFieldOf(b, harnessOf(put), cellless, cellless.Fields[1], put, nil)
	testkit.Assert(t, reason).Contains("does not name", "no cell read, nothing to stamp from")

	app := projected("Append",
		[]golang.Param{arg("ctx", ctxRef()), arg("e", namedRef("Entry"))}, []golang.Return{errRet})
	shape.ContractRoleKey("chain").Set(app.Source.EnsureMeta(), "append", "test")
	shape.ContractPartnerKey("chain", "replay").Set(app.Source.EnsureMeta(), "Nonesuch", "test")
	r := tiers.Rule{Law: lawid.AppendOnlyNoDrops, Fields: []tiers.Field{
		{Name: fReplay, Kind: tiers.KindRole, From: "chain.replay"},
		{Name: "History", Kind: tiers.KindHandle, From: handleHistoryLog},
	}}
	_, reason = lawFieldOf(b, harnessOf(app), r, r.Fields[1], app, nil)
	testkit.Assert(t, reason).Contains("is not a method", "the partner points at nothing here")
}
