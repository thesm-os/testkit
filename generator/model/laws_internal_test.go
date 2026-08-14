// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
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

// ─── The arm walk ────────────────────────────────────────────────
//
// Every refusal arm below is a header line in a generated file: the tests
// hold each to a reason that names what is missing, because a silent nil
// field is a law that runs, passes, and asserts nothing.

func namedRef(name string) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefNamed, Name: name}
}

func pkgRef(pkg, name string) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefNamed, Package: pkg, Name: name}
}

func sliceRef(elem *node.TypeRef) *node.TypeRef {
	return &node.TypeRef{TypeKind: node.TypeRefSlice, Elem: elem}
}

func ctxRef() *node.TypeRef {
	r := pkgRef("context", "Context")
	golang.MetaIsContext.Set(r.EnsureMeta(), true, "test")
	return r
}

func chanRef() *node.TypeRef {
	r := namedRef("chan")
	golang.MetaIsChannel.Set(r.EnsureMeta(), true, "test")
	return r
}

// projectedReturns is [projected] with the raw returns handed through — for
// a return whose Source carries stamps the res helper cannot spell.
func projectedReturns(name string, params []golang.Param, returns []golang.Return) *suite.Method {
	src := &node.Method{Name: name}
	return &suite.Method{Sig: &golang.Sig{
		Name: name, Params: params, Returns: returns, Source: src,
	}}
}

// projected builds a projection method by hand: the signature the arms read,
// with the classification stamps the tests choose.
func projected(name string, params []golang.Param, returns []golang.Return) *suite.Method {
	src := &node.Method{Name: name}
	return &suite.Method{Sig: &golang.Sig{
		Name: name, Params: params, Returns: returns, Source: src,
	}}
}

func arg(name string, src *node.TypeRef) golang.Param {
	return golang.Param{Name: name, Type: sdk.Builtin(src.Name), Source: src}
}

func res(src *node.TypeRef) golang.Return {
	return golang.Return{Type: sdk.Builtin(src.Name), Source: src, Error: golang.IsError(src)}
}

func stamp(m *suite.Method, shapeName, keyQ, valueQ string) *suite.Method {
	bag := m.Source.EnsureMeta()
	if shapeName != "" {
		shape.MetaShape.Set(bag, shapeName, "test")
	}
	if keyQ != "" {
		shape.MetaKeyType.Set(bag, keyQ, "test")
	}
	if valueQ != "" {
		shape.MetaValueType.Set(bag, valueQ, "test")
	}
	return m
}

// roleRule wraps one role field into the smallest rule that carries it.
func roleRule(law, field string) tiers.Rule {
	return tiers.Rule{Law: law, Fields: []tiers.Field{
		{Name: field, Kind: tiers.KindRole, From: "self"},
	}}
}

// bindField runs the role dispatch for one law/field/method triple.
func bindField(b *Bindings, law, field string, m *suite.Method) (*LawField, string) {
	r := roleRule(law, field)
	return lawFieldOf(b, nil, r, r.Fields[0], m, nil)
}

func TestRoleFieldShapes(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))

	t.Run("a scalar count binds, and its variants pick their spellings", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		field, reason := bindField(b, lawid.CountEqualsReference, "Count",
			projected("Count", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{res(namedRef("int")), errRet}))
		testkit.True(t, reason == "" && field != nil, "a nullary (int, error) counts: "+reason)

		field, reason = bindField(b, lawid.AggregatorBounded, "Read",
			projected("List", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(sliceRef(namedRef(qStr))), errRet}))
		testkit.True(t, reason == "", "a slice observation is its length: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.ScalarLen",
			"through the length adaptation")

		field, reason = bindField(b, lawid.CountEqualsReference, "Count",
			projected("Count", nil, []golang.Return{res(namedRef("int"))}))
		testkit.True(t, reason == "", "a bare scalar threads its own nil: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.ScalarNoErr",
			"through the no-error spelling")
	})

	t.Run("a scalar refuses inputs, handles, and unordered bounds", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.CountEqualsReference, "Count",
			projected("Count", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.Assert(t, reason).Contains("takes inputs", "an input needs a supplier no observation has")

		_, reason = bindField(b, lawid.CountEqualsReference, "Count",
			projected("Watch", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{res(chanRef()), errRet}))
		testkit.Assert(t, reason).Contains("identity", "a channel compares by identity")

		_, reason = bindField(b, lawid.AggregatorBounded, "Read",
			projected("Peek", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(pkgRef("example.com/x", "Payload")), errRet}))
		testkit.Assert(t, reason).Contains("no bound orders", "a bound needs an ordered scalar")
	})

	t.Run("the bare calls hold their shapes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.PredicateConsistent, "Call",
			projected("IsEmpty", nil, []golang.Return{res(namedRef("bool"))}))
		testkit.True(t, reason == "", "a bare predicate binds: "+reason)

		_, reason = bindField(b, lawid.PredicateConsistent, "Call",
			projected("IsEmpty", nil, []golang.Return{res(namedRef("bool")), errRet}))
		testkit.Assert(t, reason).Contains("not a bare predicate", "an error return is not the shape")

		_, reason = bindField(b, lawid.PureDeterministic, "Call",
			projected("Describe", nil, []golang.Return{res(namedRef(qStr))}))
		testkit.True(t, reason == "", "a bare pure call binds: "+reason)

		_, reason = bindField(b, lawid.PureDeterministic, "Call",
			projected("Describe", nil, []golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("not a bare pure call", "an error return is not the shape")
	})

	t.Run("the transformations thread one input", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		field, reason := bindField(b, lawid.TotalOver, "Call",
			projected("Classify", []golang.Param{arg("ctx", ctxRef()), arg("in", namedRef(qStr))},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.True(t, reason == "" && field.In != nil, "a one-input transformation binds: "+reason)

		_, reason = bindField(b, lawid.TotalOver, "Call",
			projected("Join", []golang.Param{arg("a", namedRef(qStr)), arg("b", namedRef(qStr))},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("several inputs", "two inputs compose no single-value closure")

		_, reason = bindField(b, lawid.TotalOver, "Call",
			projected("Sink", []golang.Param{arg("in", namedRef(qStr))}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("nothing to observe", "a void transformation observes nothing")
	})

	t.Run("the error operations hold their shapes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}, Keys: Pool{Q: qStr}}
		_, reason := bindField(b, lawid.LifecycleRespectsContext, "Op",
			projected("Close", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.True(t, reason == "", "a context-taking close binds: "+reason)

		_, reason = bindField(b, lawid.LifecycleRespectsContext, "Op",
			projected("Stop", nil, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("context-respecting",
			"the context law needs a context to cancel")

		_, reason = bindField(b, lawid.PoisonNilOnFresh, "Probe",
			projected("Err", nil, []golang.Return{errRet}))
		testkit.True(t, reason == "", "a nullary error probe binds: "+reason)

		_, reason = bindField(b, lawid.PoisonNilOnFresh, "Probe",
			projected("Check", []golang.Param{arg("k", namedRef(qStr))}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("nullary error operation", "a probe takes nothing")

		_, reason = bindField(b, lawid.LeaseDoubleAcquireBlocks, "Acquire",
			projected("Acquire", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "", "a keyed error operation binds: "+reason)

		_, reason = bindField(b, lawid.LeaseDoubleAcquireBlocks, "Acquire",
			projected("Acquire", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("keyed error operation", "an acquire takes its key")

		_, reason = bindField(b, lawid.LeaseDoubleAcquireBlocks, "Acquire",
			stamp(projected("Acquire", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef("int"))},
				[]golang.Return{errRet}), "", "int", ""))
		testkit.Assert(t, reason).Contains("beside a pool of", "the pool and the key must agree")
	})

	t.Run("the paired string write holds its shape", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.InjectionSafe, "Store",
			projected("Store", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("v", namedRef(qStr)),
			}, []golang.Return{errRet}))
		testkit.True(t, reason == "", "a string-keyed string write binds: "+reason)

		_, reason = bindField(b, lawid.InjectionSafe, "Store",
			projected("Store", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef("int")), arg("v", namedRef(qStr)),
			}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("string-keyed", "the injection probe speaks strings")
	})

	t.Run("the sum and the merge hold their shapes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.Conservative, "Sum",
			projected("Total", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.True(t, reason == "", "an integer total binds: "+reason)

		_, reason = bindField(b, lawid.Conservative, "Sum",
			projected("Name", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("no sum totals", "a string is not a conserved quantity")

		_, reason = bindField(b, lawid.Conservative, "Sum",
			projected("Total", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.Assert(t, reason).Contains("takes inputs", "a sum observes without input")

		_, reason = bindField(b, lawid.CRDTMerge, "Merge",
			projected("Merge", []golang.Param{arg("ctx", ctxRef()), arg("peer", namedRef("Replica"))},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "", "a one-peer merge binds: "+reason)

		_, reason = bindField(b, lawid.CRDTMerge, "Merge",
			projected("Merge", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("does not merge one peer", "a merge folds exactly one")
	})

	t.Run("the persisting write synthesizes its identity or refuses", func(t *testing.T) {
		t.Parallel()
		save := projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
			[]golang.Return{errRet})

		b := &Bindings{
			Subject:   suite.Subject{IfaceName: "Mixed"},
			Reference: Reference{KeyField: fieldKey},
		}
		field, reason := bindField(b, lawid.PersisterRetrievable, "Save", save)
		testkit.True(t, reason == "" && field.KeyOfName == b.KeyOfName(),
			"the saved identity is the shared projection: "+reason)

		b = &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason = bindField(b, lawid.PersisterRetrievable, "Save", save)
		testkit.Assert(t, reason).Contains("key projection", "no projection, no identity")

		_, reason = bindField(b, lawid.PersisterRetrievable, "Save",
			projected("Put", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("does not save one value", "a save takes its value")
	})

	t.Run("the offset-answering append holds its shape", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		errRet := res(namedRef("error"))
		_, reason := bindField(b, lawid.AppenderMonotonicOffsets, "Append",
			projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
				[]golang.Return{res(namedRef("int64")), errRet}))
		testkit.True(t, reason == "", "an offset-answering append binds: "+reason)

		_, reason = bindField(b, lawid.AppenderMonotonicOffsets, "Append",
			projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("answers none", "an offset is an integer")

		_, reason = bindField(b, lawid.AppenderMonotonicOffsets, "Append",
			projected("Run", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("int64")), errRet}))
		testkit.Assert(t, reason).Contains("does not append one value", "an append takes its value")
	})

	t.Run("the replay adapts a slice and refuses the rest", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		errRet := res(namedRef("error"))
		field, reason := bindField(b, lawid.AppendOnlyGrows, "Replay",
			projected("Replay", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(sliceRef(namedRef("Entry"))), errRet}))
		testkit.True(t, reason == "" && field.Out != nil, "a slice replay adapts: "+reason)

		_, reason = bindField(b, lawid.AppendOnlyGrows, "Replay",
			projected("Replay", []golang.Param{arg("ctx", ctxRef()), arg("part", namedRef(qStr))},
				[]golang.Return{res(sliceRef(namedRef("Entry"))), errRet}))
		testkit.Assert(t, reason).Contains("single-partition replay", "a partitioned replay needs the projection")

		_, reason = bindField(b, lawid.AppendOnlyGrows, "Replay",
			stamp(projected("Replay", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("Seq")), errRet}), "streamreader", "", "example.com/x.Entry"))
		testkit.Assert(t, reason).Contains("iterator", "an iterator replay is not yet composed")
	})

	t.Run("the drains pick the slice or the collect loop", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		errRet := res(namedRef("error"))
		field, reason := bindField(b, lawid.StreamNoDuplicates, "Drain",
			projected("List", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(sliceRef(namedRef(qStr))), errRet}))
		testkit.True(t, reason == "", "a slice drain binds: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.Drain", "through the slice spelling")

		_, reason = bindField(b, lawid.StreamNoDuplicates, "Drain",
			projected("List", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("Seq"))}))
		testkit.Assert(t, reason).Contains("no stamp names", "an unstamped iterator names no element")
	})

	t.Run("a rowed law's unmapped field refuses by name", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.Cacheable, "Bogus",
			projected("Get", nil, []golang.Return{res(namedRef(qStr))}))
		testkit.Assert(t, reason).Contains("transcribes no closure shape",
			"a field the table does not map is a refusal, never a guess")
	})
}

func TestValueOpField(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	keysPool := func() *Bindings {
		return &Bindings{
			Subject: suite.Subject{IfaceName: "Mixed"},
			Keys:    Pool{Field: fieldKey, Q: qStr},
			Actions: []*Action{{Pool: poolKeys}},
		}
	}

	t.Run("a single-value write binds and a chatty one refuses", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		field, reason := bindField(b, lawid.CommutativeWrite, "Write",
			projected("Apply", []golang.Param{arg("ctx", ctxRef()), arg("d", namedRef("Delta"))},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "" && field.In != nil, "one value in, one error out: "+reason)

		_, reason = bindField(b, lawid.CommutativeWrite, "Write",
			projected("Apply", []golang.Param{arg("ctx", ctxRef()), arg("d", namedRef("Delta"))},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.Assert(t, reason).Contains("more than an error", "a write answers its error alone")
	})

	t.Run("a composite write anchors on the fixture key", func(t *testing.T) {
		t.Parallel()
		b := keysPool()
		field, reason := bindField(b, lawid.IdempotentWrite, "Write",
			stamp(projected("Put", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("v", namedRef(qStr)),
			}, []golang.Return{errRet}), "", qStr, qStr))
		testkit.True(t, reason == "", "the pair anchors on the fixture key: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.WriteFixedKey", "through the anchored spelling")
		testkit.True(t, b.LawsUseFixture, "and the property now owes the fixture")

		_, reason = bindField(keysPool(), lawid.IdempotentWrite, "Write",
			stamp(projected("Put", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef("int")), arg("v", namedRef(qStr)),
			}, []golang.Return{errRet}), "", "int", qStr))
		testkit.Assert(t, reason).Contains("beside a pool of", "the anchor and the pool must agree")

		_, reason = bindField(keysPool(), lawid.IdempotentWrite, "Write",
			stamp(projected("Put", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("v", namedRef(qStr)),
			}, []golang.Return{res(namedRef(qStr)), errRet}), "", qStr, qStr))
		testkit.Assert(t, reason).Contains("more than an error", "an anchored write answers its error alone")

		_, reason = bindField(&Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}, lawid.IdempotentWrite, "Write",
			projected("Set", []golang.Param{
				arg("ctx", ctxRef()), arg("a", namedRef(qStr)),
				arg("b", namedRef(qStr)), arg("c", namedRef(qStr)),
			}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("several inputs", "three inputs compose nothing")
	})
}

// The spellings these tests repeat.
const (
	famWriter = "family.writer"
	qStr      = "string"
)

// harnessOf wraps methods into the projection lawsOf walks.
func harnessOf(methods ...*suite.Method) *suite.Contract {
	h := &suite.Contract{}
	for _, m := range methods {
		h.Methods = append(h.Methods, *m)
	}
	return h
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

func TestRoleMethodFamilies(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}, Values: Pool{Q: qStr}}

	t.Run("the writer family is the values pool's feeder", func(t *testing.T) {
		t.Parallel()
		merge := stamp(projected("Merge", []golang.Param{arg("ctx", ctxRef()), arg("p", namedRef("Replica"))},
			[]golang.Return{errRet}), "writer", "", "example.com/x.Replica")
		add := stamp(projected("Add", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef(qStr))},
			[]golang.Return{errRet}), "writer", "", qStr)

		role, reason := roleMethod(b, harnessOf(merge, add), famWriter, nil, nil)
		testkit.True(t, reason == "", "the matching writer resolves: "+reason)
		testkit.Equal(t, role.Name, "Add", "by the pool's own type, not declaration order")

		_, reason = roleMethod(b, harnessOf(merge), famWriter, nil, nil)
		testkit.Assert(t, reason).Contains("feeding the pool", "a peer-merger is not the feeder")

		loose := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		role, reason = roleMethod(loose, harnessOf(merge), famWriter, nil, nil)
		testkit.True(t, reason == "" && role.Name == "Merge",
			"with no pool to feed, the first writer stands: "+reason)
	})

	t.Run("the aggregator and cell families resolve or refuse", func(t *testing.T) {
		t.Parallel()
		agg := stamp(projected("Total", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), errRet}), "aggregator", "", "int")
		self := projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
			[]golang.Return{errRet})

		role, reason := roleMethod(b, harnessOf(agg), "family.aggregator", nil, nil)
		testkit.True(t, reason == "" && role.Name == "Total", "the aggregate resolves: "+reason)

		role, reason = roleMethod(b, harnessOf(self, agg), "family.cell", self, nil)
		testkit.True(t, reason == "" && role.Name == "Total",
			"the nullary read is the cell, the selecting writer skipped: "+reason)

		_, reason = roleMethod(b, harnessOf(self), "family.cell", self, nil)
		testkit.Assert(t, reason).Contains("no nullary read", "no cell, no cell family")
	})

	t.Run("a contract's own role and its partners resolve off the stamps", func(t *testing.T) {
		t.Parallel()
		host := projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
			[]golang.Return{errRet})
		shape.ContractRoleKey("updater").Set(host.Source.EnsureMeta(), "writer", "test")
		shape.ContractPartnerKey("updater", "reader").Set(host.Source.EnsureMeta(), "Get", "test")

		role, reason := roleMethod(b, nil, "updater.writer", host, nil)
		testkit.True(t, reason == "" && role == host, "the host fills its own role: "+reason)

		reader := projected("Get", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(namedRef("Value")), errRet})
		role, reason = roleMethod(b, harnessOf(host, reader), "updater.reader", host, nil)
		testkit.True(t, reason == "" && role.Name == "Get", "the partner stamp names the sibling: "+reason)

		_, reason = roleMethod(b, harnessOf(host), "updater.reader", host, nil)
		testkit.Assert(t, reason).Contains("not a method of", "a partner naming an absent method refuses")
	})
}

func TestGeneratorFieldArms(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	pooled := &Bindings{
		Subject: suite.Subject{IfaceName: "Mixed"},
		Keys:    Pool{Field: fieldKey, Q: qStr},
		Values:  Pool{Field: "Body", Q: "string"},
		Actions: []*Action{{Pool: poolKeys}, {Pool: poolValues}},
	}
	classify := projected("Classify",
		[]golang.Param{arg("ctx", ctxRef()), arg("in", namedRef(qStr))},
		[]golang.Return{res(namedRef(qStr)), errRet})

	genField := func(b *Bindings, law, from string, m *suite.Method, fields ...tiers.Field) (*LawField, string) {
		r := tiers.Rule{Law: law, Fields: append(fields, tiers.Field{
			Name: "Pool", Kind: tiers.KindGenerator, From: from,
		})}
		return lawFieldOf(b, nil, r, r.Fields[len(r.Fields)-1], m, nil)
	}

	t.Run("the shared pools bind where actions draw them", func(t *testing.T) {
		t.Parallel()
		field, reason := genField(pooled, lawid.Cacheable, "keys", nil)
		testkit.True(t, reason == "" && field.Pool == "keys", "the keys pool is shared: "+reason)
		field, reason = genField(pooled, lawid.Cacheable, "values", nil)
		testkit.True(t, reason == "" && field.Pool == "values", "the values pool is shared: "+reason)

		bare := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason = genField(bare, lawid.Cacheable, "keys", nil)
		testkit.Assert(t, reason).Contains("no action here declares", "an undeclared pool refuses")
		_, reason = genField(bare, lawid.Cacheable, "values", nil)
		testkit.Assert(t, reason).Contains("no action here declares", "both spellings of it")
	})

	t.Run("the law pools declare themselves once, at one type", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		roleField := tiers.Field{Name: "Call", Kind: tiers.KindRole, From: "self"}

		field, reason := genField(b, lawid.TotalOver, "inputs", classify, roleField)
		testkit.True(t, reason == "" && field.Pool == "inputs", "the inputs pool declares: "+reason)
		testkit.Equal(t, len(b.LawPools), 1, "once")

		_, reason = genField(b, lawid.TotalOver, "inputs", classify, roleField)
		testkit.True(t, reason == "", "a second law at the same type reuses it: "+reason)
		testkit.Equal(t, len(b.LawPools), 1, "still once")

		intCall := projected("Grade", []golang.Param{arg("ctx", ctxRef()), arg("in", namedRef("int"))},
			[]golang.Return{res(namedRef("int")), errRet})
		_, reason = genField(b, lawid.TotalOver, "inputs", intCall, roleField)
		testkit.Assert(t, reason).Contains("already draws", "one name, one element type")

		_, reason = genField(b, lawid.TotalOver, "inputs", nil,
			tiers.Field{Name: "Limit", Kind: tiers.KindDefault})
		testkit.Assert(t, reason).Contains("draws a domain no role here states",
			"an input pool needs a role to read its domain from")

		field, reason = genField(b, lawid.XSSSafe, "payloads", nil)
		testkit.True(t, reason == "" && field.Pool == "payloads", "the payloads pool declares: "+reason)

		_, reason = genField(b, lawid.PublisherDelivers, "messages", nil)
		testkit.Assert(t, reason).Contains("no action here declares",
			"the messages pool is the values pool, and a fixture whose sequences publish nothing has none")

		_, reason = genField(b, lawid.PublisherDelivers, "nonesuch", nil)
		testkit.Assert(t, reason).Contains("does not compose", "an unbuilt pool refuses by name")
	})
}

func TestHandleFieldArms(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}, Values: Pool{Q: qStr}}

	handle := func(b *Bindings, law, name, from string, m *suite.Method, extra ...tiers.Field) (*LawField, string) {
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
			Subject: suite.Subject{IfaceName: "Mixed"},
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

func TestConstFieldArms(t *testing.T) {
	t.Parallel()

	constField := func(law, name, from string, optional bool, m *suite.Method) (*LawField, string) {
		r := tiers.Rule{Law: law, Fields: []tiers.Field{
			{Name: name, Kind: tiers.KindConstant, From: from, Optional: optional},
		}}
		return lawFieldOf(&Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}, nil, r, r.Fields[0], m, nil)
	}

	stamped := func(key, value string) *suite.Method {
		m := unstamped()
		sdk.EnsureKey(key, sdk.StringParser).Set(m.Source.EnsureMeta(), value, "test")
		return m
	}

	t.Run("an optional constant with no stamp is omitted", func(t *testing.T) {
		t.Parallel()
		field, reason := constField(lawid.AggregatorBounded, "Min", "shape.mixin.bounded.min",
			true, unstamped())
		testkit.True(t, field == nil && reason == "", "zero is the declared floor")
	})

	t.Run("a numeric stamp renders as its literal", func(t *testing.T) {
		t.Parallel()
		field, reason := constField(lawid.AggregatorBounded, "Max", "shape.mixin.bounded.limit",
			false, stamped("shape.mixin.bounded.limit", "100"))
		testkit.True(t, reason == "" && field.Lit == "100", "the bound is the stamp's own number")
	})

	t.Run("the workflow's transitions parse or refuse", func(t *testing.T) {
		t.Parallel()
		field, reason := constField(lawid.ValidTransition, "Allowed",
			"shape.contract.workflow.param.transitions", false,
			stamped("shape.contract.workflow.param.transitions", "Draft>Live, Live>Closed"))
		testkit.True(t, reason == "", "a from>to list parses: "+reason)
		testkit.Equal(t, len(field.Pairs), 2, "into its pairs")

		_, reason = constField(lawid.ValidTransition, "Allowed",
			"shape.contract.workflow.param.transitions", false,
			stamped("shape.contract.workflow.param.transitions", "Draft"))
		testkit.Assert(t, reason).Contains("not a from>to", "an edge needs both ends")
	})

	t.Run("a contract stamp is read off any carrier", func(t *testing.T) {
		t.Parallel()
		host := unstamped()
		host.Contracts = []string{"lease"}
		sdk.EnsureKey("shape.contract.lease.param.held", sdk.StringParser).
			Set(host.Source.EnsureMeta(), "example.com/lease.ErrHeld", "test")
		sibling := unstamped()
		sibling.Contracts = []string{"lease"}

		v, ok := stampValue(harnessOf(host, sibling), sibling, "shape.contract.lease.param.held")
		testkit.True(t, ok && v == "example.com/lease.ErrHeld",
			"the host's stamp speaks for every role method")

		_, ok = stampValue(harnessOf(host), sibling, "shape.mixin.bounded.limit")
		testkit.False(t, ok, "a mixin stamp stays the selecting method's own")
	})
}

func TestIdentityCompared(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	fn := &node.TypeRef{TypeKind: node.TypeRefFunc}
	ptr := &node.TypeRef{TypeKind: node.TypeRefPointer, Elem: namedRef("Value")}

	testkit.False(t, identityCompared(projected("Total", nil,
		[]golang.Return{res(namedRef("int")), errRet})), "a value compares by value")
	testkit.True(t, identityCompared(projected("Watch", nil,
		[]golang.Return{res(chanRef()), errRet})), "a channel compares by identity")
	testkit.True(t, identityCompared(projected("Hook", nil,
		[]golang.Return{{Type: sdk.Builtin("func"), Source: fn}, errRet})),
		"a function compares by identity")
	testkit.True(t, identityCompared(projected("Find", nil,
		[]golang.Return{{Type: sdk.Builtin("ptr"), Source: ptr}, errRet})),
		"a pointer compares by identity")
	testkit.False(t, identityCompared(projected("Fire", nil, nil)),
		"nothing returned, nothing compared")
}

func TestContractParamNames(t *testing.T) {
	t.Parallel()

	testkit.True(t, len(contractParamNames("codec")) > 0,
		"a registered contract lists its parameters")
	testkit.True(t, contractParamNames("nonesuch") == nil,
		"an unregistered one lists nothing")
}

// TestClockShapedRoleFields pins the B1 shape vocabulary: the closures the
// isolation-and-clock laws render, each held to its transcription and each
// refusal to a reason a header prints.
func TestClockShapedRoleFields(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))

	t.Run("the corruption operations hold their shapes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.TamperEvident, "Tamper",
			projected("Tamper", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.True(t, reason == "", "a nullary tamper binds: "+reason)

		_, reason = bindField(b, lawid.TamperEvident, "Tamper",
			projected("Tamper", []golang.Param{arg("k", namedRef(qStr))}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("nullary error operation", "a tamper takes nothing")

		_, reason = bindField(b, lawid.PoisonConsistent, "Poison",
			projected("Poison", nil, nil))
		testkit.True(t, reason == "", "a fire-and-forget corruption binds: "+reason)

		_, reason = bindField(b, lawid.PoisonConsistent, "Poison",
			projected("Poison", []golang.Param{arg("dose", namedRef("int"))}, nil))
		testkit.Assert(t, reason).Contains("no nullary corruption", "a poison takes nothing")
	})

	t.Run("the cursor's next answers the triple", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		field, reason := bindField(b, lawid.CursorNextAfterClose, "Next",
			projected("Next", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef(qStr)), res(namedRef("bool")), errRet}))
		testkit.True(t, reason == "" && field.Out != nil, "a (value, more, error) next binds: "+reason)

		_, reason = bindField(b, lawid.CursorNextAfterClose, "Next",
			projected("Next", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("triple", "a two-return next is not the shape")
	})

	t.Run("the pinned write pins through the pool", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}, Values: Pool{Pin: fieldKey}}
		field, reason := bindField(b, lawid.TTLExpiry, "Put",
			projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "" && field.KeyField == fieldKey, "a one-value put binds on the pin: "+reason)

		_, reason = bindField(b, lawid.TTLExpiry, "Put",
			projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("v", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("one value", "a composite put is not the shape")

		unpinned := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason = bindField(unpinned, lawid.TTLExpiry, "Put",
			projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("pins nothing", "an unpinned pool cannot age a known key")
	})

	t.Run("the deadline op anchors on the projection's fixture field", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		anchored := projected("Op", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{errRet})
		anchored.ArgFields = []string{"Target"}
		field, reason := bindField(b, lawid.DeadlineRespecting, "Op", anchored)
		testkit.True(t, reason == "" && field.KeyField == "Target", "an anchored context op binds: "+reason)
		testkit.True(t, b.LawsUseFixture, "and the binding records the fixture use")

		_, reason = bindField(b, lawid.DeadlineRespecting, "Op",
			projected("Op", []golang.Param{arg("k", namedRef(qStr))}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("context operation", "the deadline needs a context to expire")

		_, reason = bindField(b, lawid.DeadlineRespecting, "Op",
			projected("Op", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("fixture field", "an unanchored input has no fixed argument")
	})

	t.Run("the scheduler's pair hold their shapes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.ScheduledFiresAfterAdvance, "Schedule",
			projected("At", []golang.Param{arg("ctx", ctxRef()), arg("after", pkgRef("time", "Duration"))},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "", "a one-offset schedule binds: "+reason)

		_, reason = bindField(b, lawid.ScheduledFiresAfterAdvance, "Schedule",
			projected("At", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("one offset", "a schedule takes its instant")

		_, reason = bindField(b, lawid.ScheduledFiresAfterAdvance, "FiredCount",
			projected("Fired", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.True(t, reason == "", "a counting observation binds: "+reason)

		_, reason = bindField(b, lawid.ScheduledFiresAfterAdvance, "FiredCount",
			projected("Fired", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.Assert(t, reason).Contains("no nullary observation", "a count takes nothing")

		_, reason = bindField(b, lawid.ScheduledFiresAfterAdvance, "FiredCount",
			projected("Fired", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("not a count", "a string is no firing tally")
	})
}

// TestClockConstAndTypeArms pins the remaining B1 arms: the duration stamp
// rendered as nanoseconds, the triple-returning result instantiation, and
// the strict value check a value-instantiating row turns on.
func TestClockConstAndTypeArms(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))

	t.Run("a duration stamp renders as untyped nanoseconds", func(t *testing.T) {
		t.Parallel()
		m := unstamped()
		sdk.EnsureKey("shape.mixin.ttl.ttl", sdk.StringParser).Set(m.Source.EnsureMeta(), "5s", "test")
		r := tiers.Rule{Law: lawid.TTLExpiry, Fields: []tiers.Field{
			{Name: "TTL", Kind: tiers.KindConstant, From: "shape.mixin.ttl.ttl"},
		}}
		field, reason := lawFieldOf(&Bindings{Subject: suite.Subject{IfaceName: "Mixed"}},
			nil, r, r.Fields[0], m, nil)
		testkit.True(t, reason == "", "a duration stamp binds: "+reason)
		testkit.Equal(t, field.Lit, "5000000000", "as nanoseconds, assignable without an import")
	})

	t.Run("a triple-returning role instantiates at its first result", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		r := roleRule(lawid.CursorNextAfterClose, "Next")
		next := projected("Next", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef(qStr)), res(namedRef("bool")), errRet})
		ref, reason := resolveArg(b, nil, r, tiers.ResultOf("Next"), next, nil)
		testkit.True(t, reason == "" && ref != nil,
			"NextOp carries the triple whole, so the type is the first result: "+reason)

		void := projected("Next", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet})
		_, reason = resolveArg(b, nil, r, tiers.ResultOf("Next"), void, nil)
		testkit.Assert(t, reason).Contains("nothing to observe",
			"a next answering only an error observes nothing")
	})

	t.Run("a value-instantiating row holds the reader to the pool's value", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject: suite.Subject{IfaceName: "Mixed"},
			Keys:    Pool{Q: qStr},
			Values:  Pool{Q: qStr, Pin: fieldKey},
		}
		mismatched := stamp(projected("Get",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(namedRef("int")), res(namedRef("error"))}), "reader", qStr, "int")
		_, reason := bindField(b, lawid.TTLExpiry, "Read", mismatched)
		testkit.Assert(t, reason).Contains("beside pools of",
			"TTLExpiry draws the value pool, so the reader must answer it")

		// Windowed's count reads (string → int) beside string pools and binds:
		// its row draws no value, so only the key is held to the pool.
		counted, reason := bindField(b, lawid.Windowed, "Count", mismatched)
		testkit.True(t, reason == "" && counted != nil,
			"a keyless row leaves the reader's value its own: "+reason)
	})

	t.Run("the template accessors answer their imports", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{}
		testkit.True(t, b.ClockPkg() != "", "the clock package is spelled for the clocked property")
		held := b.LeaseHeld()
		testkit.True(t, held.Sym == nil && held.Name == "", "no ctor error means no held sentinel")
		b.Reference.CtorErrs = []CtorErr{{Name: "ErrHeld"}}
		testkit.Equal(t, b.LeaseHeld().Name, "ErrHeld", "the first named ctor error is the held sentinel")
	})
}

// TestClockedLawBinding pins the timeaware instantiation: the ctor spelled
// from the timeaware package, the offsets pool composed bounded, and the
// Advance handle marking the binding clocked.
func TestClockedLawBinding(t *testing.T) {
	t.Parallel()

	t.Run("an Advance handle marks the binding clocked in the timeaware package", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		r := tiers.Rule{Law: lawid.ScheduledFiresAfterAdvance, Fields: []tiers.Field{
			{Name: "Advance", Kind: tiers.KindHandle, From: "clock"},
		}}
		lb, ok := lawOf(b, nil, r, nil, nil, nil)
		testkit.True(t, ok, "the handle binds")
		testkit.True(t, lb.Clocked, "and marks the law clocked")
		testkit.True(t, b.UsesClock, "so the property declares the clock")
	})

	t.Run("the offsets pool composes bounded durations", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		r := tiers.Rule{Law: lawid.ScheduledFiresAfterAdvance, Fields: []tiers.Field{
			{Name: "Offsets", Kind: tiers.KindGenerator, From: "offsets"},
		}}
		field, reason := lawFieldOf(b, nil, r, r.Fields[0], nil, nil)
		testkit.True(t, reason == "", "the pool composes: "+reason)
		testkit.Equal(t, field.Pool, "offsets", "under its own name")
		testkit.True(t, len(b.LawPools) == 1 && b.LawPools[0].Offsets,
			"and the declared pool carries the bounded-duration form")
	})
}

// TestSessionClassifierDerivation pins the per-client classifier's arms:
// every refusal names what is missing, the write-ordering laws hold out for
// a write the trace can see, and the derivation is one file-level function
// however many laws read it.
func TestSessionClassifierDerivation(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	sessionRule := func(law string) tiers.Rule {
		return tiers.Rule{Law: law, Needs: []string{mixinMonotonicReads}, Fields: []tiers.Field{
			{Name: "Classify", Kind: tiers.KindHandle, From: handleClassifier},
		}}
	}
	classify := func(b *Bindings, lawID string, m, keyed *suite.Method) (*LawField, string) {
		r := sessionRule(lawID)
		return lawFieldOf(b, nil, r, r.Fields[0], m, keyed)
	}
	stampedReader := func() *suite.Method {
		reader := projected("Get",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
		shape.MixinParamKey(mixinMonotonicReads, "version").Set(reader.Source.EnsureMeta(), "Rev", "test")
		return reader
	}

	t.Run("no keyed reader, no ordering to read", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := classify(b, lawid.MonotonicReads, unstamped(), nil)
		testkit.Assert(t, reason).Contains("no keyed reader", "the guarantee is about reads")
	})

	t.Run("no version= member, no ordering stamp", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		reader := projected("Get",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
		_, reason := classify(b, lawid.MonotonicReads, unstamped(), reader)
		testkit.Assert(t, reason).Contains("version=", "the mixin names the member or nothing orders")
	})

	t.Run("a write-ordering law holds out for a visible write", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		r := tiers.Rule{Law: lawid.MonotonicWrites, Needs: []string{"monotonicwrites"}, Fields: []tiers.Field{
			{Name: "Classify", Kind: tiers.KindHandle, From: handleClassifier},
		}}
		reader := stampedReader()
		shape.MixinParamKey("monotonicwrites", "version").Set(reader.Source.EnsureMeta(), "Rev", "test")
		_, reason := lawFieldOf(b, &suite.Contract{}, r, r.Fields[0], reader, reader)
		testkit.Assert(t, reason).Contains("answering", "the shape that would surface the stamp is named")
	})

	t.Run("the read-ordering law binds and the derivation memoizes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject:         suite.Subject{IfaceName: "Mixed"},
			Keys:            Pool{Type: sdk.Builtin(qStr)},
			sessionKeyField: fieldKey,
		}
		reader := stampedReader()
		field, reason := classify(b, lawid.MonotonicReads, reader, reader)
		testkit.True(t, reason == "" && field != nil, "a stamped keyed reader classifies: "+reason)
		testkit.Equal(t, field.KeyOfName, "mixedSessionClassify", "through the one file-level function")
		testkit.True(t, b.Session != nil && b.Session.VersionField == "Rev",
			"and the derivation lands on the bindings")

		again, reason := classify(b, lawid.MonotonicReads, reader, reader)
		testkit.True(t, reason == "" && again.KeyOfName == field.KeyOfName,
			"a second law reads the same derivation: "+reason)
	})

	t.Run("a key pool nothing draws refuses", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject:         suite.Subject{IfaceName: "Mixed"},
			sessionKeyField: fieldKey,
		}
		_, reason := classify(b, lawid.MonotonicReads, stampedReader(), stampedReader())
		testkit.Assert(t, reason).Contains("key type", "the laws instantiate at the pool's key")
	})

	t.Run("a reader with several results observes no single version", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject:         suite.Subject{IfaceName: "Mixed"},
			Keys:            Pool{Type: sdk.Builtin(qStr)},
			sessionKeyField: fieldKey,
		}
		multi := projected("Get",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(pkgRef("example.com/s", "Value")), res(namedRef("bool")), res(namedRef("error"))})
		shape.MixinParamKey(mixinMonotonicReads, "version").Set(multi.Source.EnsureMeta(), "Rev", "test")
		_, reason := classify(b, lawid.MonotonicReads, multi, multi)
		testkit.Assert(t, reason).Contains("several results", "the classifier reads one value")
	})

	t.Run("an answering writer joins the classification", func(t *testing.T) {
		t.Parallel()
		errRet := res(namedRef("error"))
		b := &Bindings{
			Subject:         suite.Subject{IfaceName: "Mixed"},
			Keys:            Pool{Type: sdk.Builtin(qStr)},
			sessionKeyField: fieldKey,
		}
		up := projected("Persist",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/s", "Value"))},
			[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
		h := &suite.Contract{Methods: []suite.Method{*up}}
		r := tiers.Rule{Law: lawid.MonotonicReads, Needs: []string{mixinMonotonicReads}, Fields: []tiers.Field{
			{Name: "Classify", Kind: tiers.KindHandle, From: handleClassifier},
		}}
		field, reason := lawFieldOf(b, h, r, r.Fields[0], stampedReader(), stampedReader())
		testkit.True(t, reason == "" && field != nil, "the spec derives beside the writer: "+reason)
		testkit.Equal(t, b.Session.Writer, "Persist", "and the write arm classifies through it")
	})

	t.Run("a value with no conventional key member refuses", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject: suite.Subject{IfaceName: "Mixed"},
			Keys:    Pool{Type: sdk.Builtin(qStr)},
		}
		_, reason := classify(b, lawid.MonotonicReads, stampedReader(), stampedReader())
		testkit.Assert(t, reason).Contains("no convention names", "per-client state needs the value's identity")
	})
}

// TestAnsweringWriterDetection pins the shape the write-ordering laws hold
// out for: one value in, the same type out beside the error.
func TestAnsweringWriterDetection(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	valueRef := pkgRef("example.com/s", "Value")

	up := projected("Persist",
		[]golang.Param{arg("ctx", ctxRef()), arg("v", valueRef)},
		[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
	plain := projected("Put",
		[]golang.Param{arg("ctx", ctxRef()), arg("v", valueRef)},
		[]golang.Return{errRet})
	crossed := projected("Save",
		[]golang.Param{arg("ctx", ctxRef()), arg("v", valueRef)},
		[]golang.Return{res(namedRef("int64")), errRet})

	h := &suite.Contract{Methods: []suite.Method{*plain, *up, *crossed}}
	found := answeringWriterOf(h)
	testkit.True(t, found != nil && found.Name == "Persist",
		"the answered-state write is the answering writer")

	none := &suite.Contract{Methods: []suite.Method{*plain, *crossed}}
	testkit.True(t, answeringWriterOf(none) == nil,
		"an error-only write and a scalar-answering write both hide the stored state")
}

// TestSessionVersionScan pins the stamp scan the twin decision reads: the
// first session mixin carrying version= answers, anything else scans past.
func TestSessionVersionScan(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	reader := projected("Get",
		[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
		[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
	reader.Mixins = []string{"monotonicreads"}
	shape.MixinParamKey(mixinMonotonicReads, "version").Set(reader.Source.EnsureMeta(), "Rev", "test")

	carrier, member, stamped := sessionVersionOf(&suite.Contract{Methods: []suite.Method{*reader}})
	testkit.True(t, stamped && member == "Rev", "a stamped session mixin names its member")
	testkit.True(t, carrier != nil && carrier.Name == "Get", "and the carrying method rides along")

	bare := projected("Get",
		[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
		[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
	bare.Mixins = []string{"monotonicreads"}
	_, _, stamped = sessionVersionOf(&suite.Contract{Methods: []suite.Method{*bare}})
	testkit.False(t, stamped, "a session mixin without version= stamps no ordering")

	other := projected("Put",
		[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef(qStr))},
		[]golang.Return{errRet})
	other.Mixins = []string{"idempotent"}
	_, _, stamped = sessionVersionOf(&suite.Contract{Methods: []suite.Method{*other}})
	testkit.False(t, stamped, "a non-session mixin is not in the scan")
}

// TestSessionConcurrentDerivation pins the session leg: it derives whole —
// reader and writer both driven — or not at all, and lands the stepless
// family the trace laws carry.
func TestSessionConcurrentDerivation(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	keyed := projected("Get",
		[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
		[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
	valued := projected("Persist",
		[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/s", "Value"))},
		[]golang.Return{errRet})

	b := &Bindings{
		Subject: suite.Subject{IfaceName: "Mixed"},
		Session: &SessionSpec{ClassifyName: "mixedSessionClassify"},
		Actions: []*Action{{Method: "Get"}, {Method: "Persist"}},
	}
	concurrentOf(b, &suite.Contract{}, keyed, valued)
	testkit.Equal(t, b.ConcFamily, "session", "both halves in hand derive the stepless leg")

	half := &Bindings{
		Subject: suite.Subject{IfaceName: "Mixed"},
		Session: &SessionSpec{ClassifyName: "mixedSessionClassify"},
		Actions: []*Action{{Method: "Get"}},
	}
	concurrentOf(half, &suite.Contract{}, keyed, valued)
	testkit.Equal(t, half.ConcFamily, "", "half a pair interleaves nothing worth checking")
	testkit.True(t, half.ConcReader == nil && half.ConcWriter == nil,
		"and the halves are reset rather than left dangling")
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
	subscribeWith := func(ret golang.Return) suite.Method {
		return *projectedReturns("Subscribe",
			[]golang.Param{arg("ctx", ctxRef())}, []golang.Return{ret, errRet})
	}
	carrier := func() *suite.Method {
		m := projected("Publish",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/p", "Value"))},
			[]golang.Return{errRet})
		shape.ContractPartnerKey("publisher", "subscribe").Set(m.Source.EnsureMeta(), "Subscribe", "test")
		return m
	}

	t.Run("a channel-answering subscribe derives the sweep once", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}
		h := &suite.Contract{Methods: []suite.Method{subscribeWith(chanReturn())}}
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
		b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}
		h := &suite.Contract{Methods: []suite.Method{subscribeWith(res(pkgRef("example.com/p", "Handle")))}}
		_, reason := lawFieldOf(b, h, drainRule, drainRule.Fields[0], carrier(), nil)
		testkit.Assert(t, reason).Contains("no channel", "an object handle is the drain option's territory")
	})

	t.Run("a carrier that stamps no subscribe partner refuses", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}
		h := &suite.Contract{Methods: []suite.Method{subscribeWith(chanReturn())}}
		unstampedCarrier := projected("Publish",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/p", "Value"))},
			[]golang.Return{res(namedRef("error"))})
		_, reason := lawFieldOf(b, h, drainRule, drainRule.Fields[0], unstampedCarrier, nil)
		testkit.Assert(t, reason).Contains("does not stamp", "the partner is the directive's to name")
	})
}

// TestPublisherModeConstant pins the mode spelling map: the three directive
// spellings land on the engine enum, and anything else refuses by name.
func TestPublisherModeConstant(t *testing.T) {
	t.Parallel()

	modeField := func(law, value string) (*LawField, string) {
		r := tiers.Rule{Law: law, Fields: []tiers.Field{
			{Name: "Mode", Kind: tiers.KindConstant, From: "shape.contract.publisher.param.mode"},
		}}
		m := unstamped()
		sdk.EnsureKey("shape.contract.publisher.param.mode", sdk.StringParser).
			Set(m.Source.EnsureMeta(), value, "test")
		return lawFieldOf(&Bindings{Subject: suite.Subject{IfaceName: "Contract"}}, nil, r, r.Fields[0], m, nil)
	}

	field, reason := modeField(lawid.PublisherAtLeastOnce, "at-least-once")
	testkit.True(t, reason == "" && field.Const != nil, "the at-least bound spells its enum: "+reason)
	_, reason = modeField(lawid.PublisherAtMostOnce, "at-most-once")
	testkit.True(t, reason == "", "the at-most bound spells its enum: "+reason)
	_, reason = modeField(lawid.PublisherExactlyOnce, "exactly-once")
	testkit.True(t, reason == "", "the exactly bound spells its enum: "+reason)
	_, reason = modeField(lawid.PublisherExactlyOnce, "sometimes")
	testkit.Assert(t, reason).Contains("not a delivery mode", "an unknown spelling refuses by name")
}

// TestSubscribeShape pins the subscription closure: the handle is kept for
// the drain, never compared, and the shape takes nothing.
func TestSubscribeShape(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}
	field, reason := bindField(b, lawid.PublisherDelivers, "Subscribe",
		projected("Subscribe", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("example.com/p", "Handle")), errRet}))
	testkit.True(t, reason == "" && field.Out != nil, "a nullary subscription binds: "+reason)

	_, reason = bindField(b, lawid.PublisherDelivers, "Subscribe",
		projected("Subscribe", []golang.Param{arg("ctx", ctxRef()), arg("topic", namedRef(qStr))},
			[]golang.Return{res(pkgRef("example.com/p", "Handle")), errRet}))
	testkit.Assert(t, reason).Contains("no subscription draw supplies", "a topic is an input nothing draws")
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
			Subject: suite.Subject{IfaceName: "Contract"},
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
		b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}
		b.LawPools = append(b.LawPools, LawPool{Name: "payloads", Q: "int", Elem: sdk.Builtin("int")})
		r := tiers.Rule{Law: lawid.XSSSafe, Fields: []tiers.Field{
			{Name: "Payloads", Kind: tiers.KindGenerator, From: "payloads"},
		}}
		_, reason := lawFieldOf(b, nil, r, r.Fields[0], nil, nil)
		testkit.True(t, reason != "", "two laws asking one name at two types are caught")

		b2 := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}
		b2.LawPools = append(b2.LawPools, LawPool{Name: "offsets", Q: builtin64, Elem: sdk.Builtin(builtin64)})
		r2 := tiers.Rule{Law: lawid.ScheduledFiresAfterAdvance, Fields: []tiers.Field{
			{Name: "Offsets", Kind: tiers.KindGenerator, From: "offsets"},
		}}
		_, reason = lawFieldOf(b2, nil, r2, r2.Fields[0], nil, nil)
		testkit.True(t, reason != "", "the offsets pool holds one type too")
	})

	t.Run("a subscription answering nothing refuses the drain", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}
		bare := projectedReturns("Subscribe", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet})
		h := &suite.Contract{Methods: []suite.Method{*bare}}
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
		b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}
		ch := namedRef("chan")
		golang.MetaIsChannel.Set(ch.EnsureMeta(), true, "test")
		sub := projectedReturns("Subscribe", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{{Type: sdk.Builtin("sub"), Source: ch}, errRet})
		h := &suite.Contract{Methods: []suite.Method{*sub}}
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

// TestSuppliedDoors pins the door builder's arms: each shape spells its
// closure at the fixture's types or refuses with what is missing, a field
// shared by several laws builds one door, and a name asked at two shapes is
// a conflict rather than a shadow.
func TestSuppliedDoors(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	door := func(b *Bindings, law, field, from string, m *suite.Method) (*LawField, string) {
		r := tiers.Rule{Law: law, Fields: []tiers.Field{
			{Name: field, Kind: tiers.KindSupplied, From: from},
		}}
		return lawFieldOf(b, nil, r, r.Fields[0], m, nil)
	}

	t.Run("a key-typed door needs the keys pool", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := door(b, lawid.CausalOrdering, "HappensBefore", "happens-before", nil)
		testkit.Assert(t, reason).Contains("key no method", "ClientOp is keyed")

		b.Keys = Pool{Type: sdk.Builtin(qStr)}
		field, reason := door(b, lawid.CausalOrdering, "HappensBefore", "happens-before", nil)
		testkit.True(t, reason == "" && field.Pool == "happensBefore",
			"the door opens at the pool's key: "+reason)
	})

	t.Run("an element-typed door reads the drained slice", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
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
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}, Keys: Pool{Type: sdk.Builtin(qStr)}}
		_, reason := door(b, lawid.SnapshotIsolationG0, "History", "history", nil)
		testkit.True(t, reason == "", "the first isolation level opens the door: "+reason)
		_, reason = door(b, lawid.SnapshotIsolationG1, "History", "history", nil)
		testkit.True(t, reason == "", "the second reads the same one: "+reason)
		testkit.Equal(t, len(b.SuppliedOptions), 1, "one door, three laws")
	})

	t.Run("a name asked at two shapes is a conflict", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		testkit.Equal(t, b.addSuppliedOption(&SuppliedOption{Config: "x", Shape: supSubjPred}), "",
			"the first spelling lands")
		testkit.Assert(t, b.addSuppliedOption(&SuppliedOption{Config: "x", Shape: supStats})).
			Contains("second shape", "and the second is a conflict, not a shadow")
	})

	t.Run("the subject-only doors open unconditionally", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := door(b, lawid.PoolLeakFree, "Balanced", "balanced", nil)
		testkit.True(t, reason == "", "the balance door: "+reason)
		_, reason = door(b, lawid.PoolBalanced, "Stats", "stats", nil)
		testkit.True(t, reason == "", "the stats door: "+reason)
		free := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}, Keys: Pool{Type: sdk.Builtin(qStr)}}
		_, reason = door(free, lawid.LeaseReleasedOnCancel, "Free", "free", nil)
		testkit.True(t, reason == "", "the free door: "+reason)
	})

	t.Run("the merge door reads the observation", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := door(b, lawid.EventualConvergence, "Merge", "merge", nil)
		testkit.Assert(t, reason).Contains("observes state through no method",
			"no observation, no lattice to join")

		agg := projected("Count", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), res(namedRef("error"))})
		stamp(agg, "aggregator", "", "")
		h := &suite.Contract{Methods: []suite.Method{*agg}}
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
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		replay := projected("Replay",
			[]golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(sliceRef(pkgRef("example.com/c", "Entry"))), errRet})
		carrier := projected("Append",
			[]golang.Param{arg("ctx", ctxRef()), arg("e", pkgRef("example.com/c", "Entry"))},
			[]golang.Return{errRet})
		shape.ContractPartnerKey("chain", "replay").Set(carrier.Source.EnsureMeta(), "Replay", "test")
		h := &suite.Contract{Methods: []suite.Method{*replay, *carrier}}
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
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := door(b, lawid.ReplayCausalOrdering, fEntryID, "entry-id", unstamped())
		testkit.True(t, reason != "", "no chain.replay stamp, no entry to identify")
	})

	t.Run("a field the table does not transcribe keeps the refusal", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := door(b, lawid.ReadAfterWrite, "Nonesuch", "nonesuch", nil)
		testkit.Assert(t, reason).Contains("no generated value can stand in for",
			"an untranscribed field is not a door")
	})
}

// funcRef builds a func-kinded type ref — the callable a compute-taking or
// body-taking role declares.
func funcRef() *node.TypeRef {
	return &node.TypeRef{Name: "func", TypeKind: node.TypeRefFunc}
}

// TestContractShapeArms pins the contract-shape closure family: a
// handle-answering begin, the terminal pair that threads it, the saga's
// coordinating run, the compute-taking call, the failing-body run and the
// page-shaped walk — each binding at its fixture's types or refusing with
// what is missing.
func TestContractShapeArms(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}

	t.Run("a handle-answering begin binds and a flat one refuses", func(t *testing.T) {
		t.Parallel()
		field, reason := bindField(b, lawid.TwoPhaseMutex, "Begin",
			projected("Begin", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("Tx")), errRet}))
		testkit.True(t, reason == "" && field.Out != nil, "the handle draw binds: "+reason)

		_, reason = bindField(b, lawid.TwoPhaseMutex, "Begin",
			projected("Begin", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("answers no handle", "a flat begin threads nothing")

		_, reason = bindField(b, lawid.TwoPhaseMutex, "Begin",
			projected("Begin", []golang.Param{arg("ctx", ctxRef()), arg("opts", namedRef(qStr))},
				[]golang.Return{res(namedRef("Tx")), errRet}))
		testkit.Assert(t, reason).Contains("no handle draw supplies", "an input nothing draws")
	})

	// The staging write a mid-transaction claim is stated about. Bound from
	// the interface rather than supplied as a closure, because a supplied one
	// reaches past to the concrete store and no defect worn on the interface
	// can then reach the law — bound, green, and unfalsifiable.
	t.Run("a staging write threads the handle beside a key and a value", func(t *testing.T) {
		t.Parallel()
		staging := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}

		field, reason := bindField(staging, lawid.TransactionNoMidTxVisibility, "TxPut",
			projected("Put", []golang.Param{
				arg("ctx", ctxRef()), arg("h", namedRef("Tx")),
				arg("k", namedRef(qStr)), arg("v", namedRef("Value")),
			}, []golang.Return{errRet}))
		testkit.True(t, reason == "", "the staging write binds: "+reason)
		testkit.True(t, field != nil && field.In != nil && field.Key != nil && field.Value != nil,
			"and carries the handle, the key and the value it writes")

		_, reason = bindField(staging, lawid.TransactionNoMidTxVisibility, "TxPut",
			projected("Put", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("v", namedRef("Value")),
			}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("inside an open handle",
			"a keyed write with no handle stages nothing a transaction owns")
	})

	t.Run("a terminal operation threads begin's handle", func(t *testing.T) {
		t.Parallel()
		begin := projected("Begin", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("Tx")), errRet})
		shape.ContractRoleKey("tx").Set(begin.Source.EnsureMeta(), "begin", "test")
		shape.ContractPartnerKey("tx", "commit").Set(begin.Source.EnsureMeta(), "Commit", "test")
		commit := projected("Commit",
			[]golang.Param{arg("ctx", ctxRef()), arg("h", namedRef("Tx"))}, []golang.Return{errRet})
		r := tiers.Rule{Law: lawid.TwoPhaseMutex, Fields: []tiers.Field{
			{Name: "Begin", Kind: tiers.KindRole, From: "tx.begin"},
			{Name: "Commit", Kind: tiers.KindRole, From: "tx.commit"},
		}}
		field, reason := lawFieldOf(b, harnessOf(begin, commit), r, r.Fields[1], begin, nil)
		testkit.True(t, reason == "" && field.In != nil, "the terminal op binds: "+reason)

		stranger := projected("Commit",
			[]golang.Param{arg("ctx", ctxRef()), arg("h", namedRef("Other"))}, []golang.Return{errRet})
		_, reason = lawFieldOf(b, harnessOf(begin, stranger), r, r.Fields[1], begin, nil)
		testkit.Assert(t, reason).Contains("settles a", "a stranger handle is named, not compiled against")
	})

	t.Run("the saga run closes over drawn steps and its compensation", func(t *testing.T) {
		t.Parallel()
		step := projected("Step",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))}, []golang.Return{errRet})
		shape.ContractRoleKey("saga").Set(step.Source.EnsureMeta(), "step", "test")
		shape.ContractPartnerKey("saga", "compensate").Set(step.Source.EnsureMeta(), "Compensate", "test")
		comp := projected("Compensate",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))}, []golang.Return{errRet})
		r := tiers.Rule{Law: lawid.SagaFullCompensation, Fields: []tiers.Field{
			{Name: "Run", Kind: tiers.KindRole, From: "saga.step"},
		}}

		pooled := &Bindings{
			Subject: suite.Subject{IfaceName: "Contract"},
			Values:  Pool{Type: sdk.Builtin("Value"), Q: "Value"},
			Actions: []*Action{{Method: "Step", Pool: poolValues}},
		}
		field, reason := lawFieldOf(pooled, harnessOf(step, comp), r, r.Fields[0], step, nil)
		testkit.True(t, reason == "", "the coordinating run binds: "+reason)
		testkit.Equal(t, field.Partner, "Compensate", "unwinding through the pinned pairing")
		testkit.Equal(t, field.Pool, poolValues, "over the drawn steps")

		_, reason = lawFieldOf(b, harnessOf(step, comp), r, r.Fields[0], step, nil)
		testkit.Assert(t, reason).Contains("no action here declares", "no pool, no steps to draw")
	})

	t.Run("a compute-taking call binds and a computeless one refuses", func(t *testing.T) {
		t.Parallel()
		field, reason := bindField(b, lawid.SingleflightCoalesces, fCall,
			projected("Run",
				[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr)), arg("compute", funcRef())},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.True(t, reason == "" && field.Key != nil && field.Out != nil,
			"the deduplicated call binds: "+reason)

		_, reason = bindField(b, lawid.SingleflightCoalesces, fCall,
			projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("no compute to deduplicate", "nothing to count")
	})

	t.Run("a failing-body run binds and a bodyless one refuses", func(t *testing.T) {
		t.Parallel()
		field, reason := bindField(b, lawid.TransactionRollback, "Run",
			projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("body", funcRef())},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "" && field != nil, "the scoped run binds: "+reason)

		_, reason = bindField(b, lawid.TransactionRollback, "Run",
			projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("accepts no failing body", "a keyed run cannot be made to fail")
	})

	t.Run("a page-shaped read binds and each missing half refuses", func(t *testing.T) {
		t.Parallel()
		page := func(next, more *node.TypeRef) *suite.Method {
			return projected("Page",
				[]golang.Param{arg("ctx", ctxRef()), arg("cur", namedRef("Cursor"))},
				[]golang.Return{res(sliceRef(namedRef("Value"))), res(next), res(more), errRet})
		}
		field, reason := bindField(b, lawid.PaginatorNoDuplicates, "Page",
			page(namedRef("Cursor"), namedRef("bool")))
		testkit.True(t, reason == "" && field.In != nil && field.Out != nil,
			"the page walk binds: "+reason)

		_, reason = bindField(b, lawid.PaginatorNoDuplicates, "Page",
			projected("Page", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{res(namedRef("Value")), errRet}))
		testkit.Assert(t, reason).Contains("no cursor to resume from", "a keyed read walks nothing")

		_, reason = bindField(b, lawid.PaginatorNoDuplicates, "Page",
			page(namedRef(qStr), namedRef("bool")))
		testkit.Assert(t, reason).Contains("which is not its", "the next cursor resumes at another type")

		_, reason = bindField(b, lawid.PaginatorNoDuplicates, "Page",
			page(namedRef("Cursor"), namedRef("int")))
		testkit.Assert(t, reason).Contains("never says whether more remains", "no termination signal")
	})
}

// TestCoalesceHandlesAndIdentityKey pins the two handle arms the family
// added: the probe/counter pair the coalescing law instruments itself with,
// and the key projection's identity fallback for the walk.
func TestCoalesceHandlesAndIdentityKey(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}
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

// TestContractShapeRefusalClauses walks the family's remaining refusal
// clauses one by one — each is a distinct wrong fixture a named refusal must
// meet, and a clause nothing exercises is a message nothing proofread.
func TestContractShapeRefusalClauses(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}

	t.Run("a terminal pair's own wrong shapes refuse", func(t *testing.T) {
		t.Parallel()
		begin := projected("Begin", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("Tx")), errRet})
		shape.ContractRoleKey("tx").Set(begin.Source.EnsureMeta(), "begin", "test")
		shape.ContractPartnerKey("tx", "commit").Set(begin.Source.EnsureMeta(), "Commit", "test")
		r := tiers.Rule{Law: lawid.TwoPhaseMutex, Fields: []tiers.Field{
			{Name: fBegin, Kind: tiers.KindRole, From: "tx.begin"},
			{Name: fCommit, Kind: tiers.KindRole, From: "tx.commit"},
		}}

		wide := projected("Commit",
			[]golang.Param{arg("ctx", ctxRef()), arg("h", namedRef("Tx")), arg("note", namedRef(qStr))},
			[]golang.Return{errRet})
		_, reason := lawFieldOf(b, harnessOf(begin, wide), r, r.Fields[1], begin, nil)
		testkit.Assert(t, reason).Contains("does not settle one handle", "two inputs settle nothing")

		flatBegin := projected("Begin", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet})
		shape.ContractRoleKey("tx").Set(flatBegin.Source.EnsureMeta(), "begin", "test")
		shape.ContractPartnerKey("tx", "commit").Set(flatBegin.Source.EnsureMeta(), "Commit", "test")
		commit := projected("Commit",
			[]golang.Param{arg("ctx", ctxRef()), arg("h", namedRef("Tx"))}, []golang.Return{errRet})
		_, reason = lawFieldOf(b, harnessOf(flatBegin, commit), r, r.Fields[1], flatBegin, nil)
		testkit.Assert(t, reason).Contains("answers none", "no handle to thread")
	})

	t.Run("the saga run's own wrong shapes refuse", func(t *testing.T) {
		t.Parallel()
		r := tiers.Rule{Law: lawid.SagaFullCompensation, Fields: []tiers.Field{
			{Name: fRun, Kind: tiers.KindRole, From: "saga.step"},
		}}
		pooled := &Bindings{
			Subject: suite.Subject{IfaceName: "Contract"},
			Values:  Pool{Type: sdk.Builtin("Value"), Q: "Value"},
			Actions: []*Action{{Method: "Step", Pool: poolValues}},
		}

		wide := projected("Step",
			[]golang.Param{arg("ctx", ctxRef()), arg("a", namedRef("Value")), arg("b", namedRef("Value"))},
			[]golang.Return{errRet})
		shape.ContractRoleKey("saga").Set(wide.Source.EnsureMeta(), "step", "test")
		_, reason := lawFieldOf(pooled, harnessOf(wide), r, r.Fields[0], wide, nil)
		testkit.Assert(t, reason).Contains("does not step one value", "two inputs step nothing")

		stranger := stamp(projected("Step",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Other"))},
			[]golang.Return{errRet}), "", "", "Other")
		shape.ContractRoleKey("saga").Set(stranger.Source.EnsureMeta(), "step", "test")
		_, reason = lawFieldOf(pooled, harnessOf(stranger), r, r.Fields[0], stranger, nil)
		testkit.Assert(t, reason).Contains("beside a pool of", "steps a type the pool never draws")

		orphan := projected("Step",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))}, []golang.Return{errRet})
		shape.ContractRoleKey("saga").Set(orphan.Source.EnsureMeta(), "step", "test")
		_, reason = lawFieldOf(pooled, harnessOf(orphan), r, r.Fields[0], orphan, nil)
		testkit.Assert(t, reason).Contains("does not stamp", "no compensation, no unwinding")
	})

	t.Run("a compute of the wrong kind refuses", func(t *testing.T) {
		t.Parallel()
		_, reason := bindField(b, lawid.SingleflightCoalesces, fCall,
			projected("Run",
				[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr)), arg("note", namedRef(qStr))},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("no compute to deduplicate", "a second string is not a callable")

		_, reason = bindField(b, lawid.SingleflightCoalesces, fCall,
			projected("Run",
				[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr)), arg("compute", funcRef())},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("nothing to observe", "a call answering nothing shares nothing")
	})

	t.Run("a body-run of the wrong arity refuses", func(t *testing.T) {
		t.Parallel()
		_, reason := bindField(b, lawid.TransactionRollback, fRun,
			projected("Run",
				[]golang.Param{arg("ctx", ctxRef()), arg("body", funcRef()), arg("note", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("accepts no failing body", "two inputs scope nothing")
	})

	t.Run("a page walk with no error or no cursor refuses", func(t *testing.T) {
		t.Parallel()
		_, reason := bindField(b, lawid.PaginatorNoDuplicates, fPage,
			projected("Page",
				[]golang.Param{arg("ctx", ctxRef()), arg("cur", namedRef("Cursor"))},
				[]golang.Return{
					res(sliceRef(namedRef("Value"))), res(namedRef("Cursor")), res(namedRef("bool")),
				}))
		testkit.Assert(t, reason).Contains("no cursor to resume from", "three results are not a page")

		_, reason = bindField(b, lawid.PaginatorNoDuplicates, fPage,
			projected("Page", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{
					res(sliceRef(namedRef("Value"))), res(namedRef("Cursor")), res(namedRef("bool")), errRet,
				}))
		testkit.Assert(t, reason).Contains("no cursor to resume from", "nothing to resume by")
	})

	t.Run("a probe over an answerless call refuses", func(t *testing.T) {
		t.Parallel()
		bare := projected("Run",
			[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr)), arg("compute", funcRef())},
			[]golang.Return{errRet})
		r := tiers.Rule{Law: lawid.SingleflightCoalesces, Fields: []tiers.Field{
			{Name: fCall, Kind: tiers.KindRole, From: fromSelf},
			{Name: "Compute", Kind: tiers.KindHandle, From: handleCoalesce},
		}}
		_, reason := lawFieldOf(b, nil, r, r.Fields[1], bare, nil)
		testkit.Assert(t, reason).Contains("nothing to observe", "no answer, nothing for the probe to type")
	})
}

// TestContractShapeAnswerClauses covers the family's answer-shaped
// refusals: a begin answering two things threads neither, and a body-run
// answering more than an error is not the scope the law forwards into.
func TestContractShapeAnswerClauses(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}

	_, reason := bindField(b, lawid.TwoPhaseMutex, fBegin,
		projected("Begin", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("Tx")), res(namedRef("Lease")), errRet}))
	testkit.Assert(t, reason).Contains("answers no handle", "two answers thread neither")

	_, reason = bindField(b, lawid.TransactionRollback, fRun,
		projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("body", funcRef())},
			[]golang.Return{res(namedRef(qStr)), errRet}))
	testkit.Assert(t, reason).Contains("accepts no failing body", "an answering run is another shape")
}

// TestVersionStampAndHistoryHandles pins the small pair's two handles: the
// version-coherent draw over the cas cell, and the append-recording history
// the no-drops law reads.
func TestVersionStampAndHistoryHandles(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}

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
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}

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
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}

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

// TestWatcherMemberClosures pins the member-scope wiring: the keyed handle
// draw, the keyed write beside it, and the two closures derived from the
// next=/stop= member stamps — each binding at the fixture's types or
// refusing with what is missing.
func TestWatcherMemberClosures(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{
		Subject: suite.Subject{IfaceName: "Contract"},
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
	b := &Bindings{Subject: suite.Subject{IfaceName: "Contract"}}

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

// TestReplicaClosureShapes pins the multi-replica family: the pairwise sync
// composed into the star round, the per-replica settle, and the refusals a
// wrong peer or a wide settle earns.
func TestReplicaClosureShapes(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}

	syncRole := projected("Sync",
		[]golang.Param{arg("ctx", ctxRef()), arg("peer", namedRef("Mixed"))}, []golang.Return{errRet})
	field, reason := bindField(b, lawid.EventualConvergence, "Sync", syncRole)
	testkit.True(t, reason == "" && field != nil, "the pairwise sync binds: "+reason)

	stranger := projected("Sync",
		[]golang.Param{arg("ctx", ctxRef()), arg("peer", namedRef("Other"))}, []golang.Return{errRet})
	_, reason = bindField(b, lawid.EventualConvergence, "Sync", stranger)
	testkit.Assert(t, reason).Contains("syncs with a Other", "a foreign peer is named, not guessed")

	wide := projected("Sync",
		[]golang.Param{arg("ctx", ctxRef()), arg("a", namedRef("Mixed")), arg("b", namedRef("Mixed"))},
		[]golang.Return{errRet})
	_, reason = bindField(b, lawid.EventualConvergence, "Sync", wide)
	testkit.Assert(t, reason).Contains("does not sync with one peer", "two peers are another protocol")

	settle := projected("Settle", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet})
	field, reason = bindField(b, lawid.EventualConvergence, "Settle", settle)
	testkit.True(t, reason == "" && field != nil, "the per-replica settle binds: "+reason)

	wideSettle := projected("Settle",
		[]golang.Param{arg("ctx", ctxRef()), arg("n", namedRef("int"))}, []golang.Return{errRet})
	_, reason = bindField(b, lawid.EventualConvergence, "Settle", wideSettle)
	testkit.Assert(t, reason).Contains("not a nullary settle", "a parameterised settle draws nothing")
}

// TestMidTxDoorAndReadbackPool pins the mid-transaction wiring: the TxPut
// door spelled at the handle, key and read-back types, the readback pool at
// the observed reader's answer, and the refusals each earns without its
// anchors.
func TestMidTxStagingAndReadbackPool(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	begin := projected("Begin", []golang.Param{arg("ctx", ctxRef())},
		[]golang.Return{res(namedRef("Tx")), errRet})
	shape.ContractRoleKey("tx").Set(begin.Source.EnsureMeta(), "begin", "test")
	get := stamp(projected("Get",
		[]golang.Param{arg("ctx", ctxRef()), arg("key", namedRef(qStr))},
		[]golang.Return{res(namedRef("Value")), errRet}), "reader", qStr, "Value")
	// The staging write is derived from the interface now, not supplied. A
	// door reaching past to the concrete store is what made this law
	// unfalsifiable: every mutant the prover wore failed the type assertion
	// inside the consumer's closure before the law observed anything.
	put := projected("Put", []golang.Param{
		arg("ctx", ctxRef()), arg("h", namedRef("Tx")),
		arg("key", namedRef(qStr)), arg("v", namedRef("Value")),
	}, []golang.Return{errRet})
	r := tiers.Rule{Law: lawid.TransactionNoMidTxVisibility, Fields: []tiers.Field{
		{Name: fBegin, Kind: tiers.KindRole, From: "tx.begin"},
		{Name: fTxPut, Kind: tiers.KindRole, From: "family.handlewriter"},
		{Name: fRead, Kind: tiers.KindRole, From: "family.reader"},
		{Name: "Values", Kind: tiers.KindGenerator, From: "readback"},
	}}

	t.Run("the staging write and the pool bind at the fixture's types", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject: suite.Subject{IfaceName: "Contract"},
			Keys:    Pool{Type: sdk.Builtin(qStr), Q: qStr, Field: "Key"},
		}
		field, reason := lawFieldOf(b, harnessOf(begin, put, get), r, r.Fields[1], begin, get)
		testkit.True(t, reason == "", "the staging write binds: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.HandleWrite",
			"as a closure over a real method, not a door")

		field, reason = lawFieldOf(b, harnessOf(begin, put, get), r, r.Fields[3], begin, get)
		testkit.True(t, reason == "", "the readback pool binds: "+reason)
		testkit.Equal(t, field.Pool, "readback", "at the observed reader's answer")
	})

	t.Run("each anchor's absence refuses by name", func(t *testing.T) {
		t.Parallel()
		pooled := &Bindings{
			Subject: suite.Subject{IfaceName: "Contract"},
			Keys:    Pool{Type: sdk.Builtin(qStr), Q: qStr, Field: "Key"},
		}
		_, reason := lawFieldOf(pooled, harnessOf(begin, get), r, r.Fields[1], begin, get)
		testkit.Assert(t, reason).Contains("no write threading an open handle",
			"nothing on the interface stages, so the claim declines")

		_, reason = lawFieldOf(pooled, harnessOf(begin), r, r.Fields[3], begin, nil)
		testkit.Assert(t, reason).Contains("no keyed reader", "no read-back, no domain to draw")
	})
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
		read.Mixins = []string{"deleteremoves"}
		shape.MixinParamKey("deleteremoves", "sentinel").
			Set(read.Source.EnsureMeta(), "example.com/dr.ErrGone", "test")
		sym := missSentinelOf(harnessOf(read))
		testkit.True(t, sym != nil, "the stamped sentinel is the oracle's miss")

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
			Subject:   suite.Subject{IfaceName: "Mixed"},
			Values:    Pool{Type: sdk.Builtin("Value"), Q: "Value"},
			Keys:      Pool{Type: sdk.Builtin(qStr), Q: qStr, Field: "Key"},
			Actions:   []*Action{{Method: "Store", Pool: poolValues}, {Method: "Get", Pool: poolKeys}},
			Reference: Reference{KeyField: "Key"},
		}
		field := &LawField{BaseEmit: b.BaseEmit, Name: "Disturb", Iface: b.IfaceRef, Key: b.Keys.Type}
		got, reason := disturbFieldOf(b, harnessOf(writer), field, nil, nil)
		testkit.True(t, reason == "" && got != nil, "the writer-fed disturbance binds: "+reason)
		testkit.Equal(t, string(got.Kind()), "model.lawfield.DisturbWrite", "as the adjacent-key write")

		keyless := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		omitted, reason := disturbFieldOf(keyless, harnessOf(writer), field, nil, nil)
		testkit.True(t, omitted == nil && reason == "",
			"no pools, no projection — the field stays omitted, never guessed")
	})
}

// TestSaturationDerivation pins the prover's own derivation: the wardrobe's
// kinds per method shape, the session laws' reachable pair, the unwearable
// skip, and the one law whose kill criterion is the differential itself.
func TestSaturationDerivation(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))

	t.Run("the kinds follow the method's shape", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject: suite.Subject{IfaceName: "Mixed"},
			Values:  Pool{Type: sdk.Builtin("Value"), Q: "Value", Field: "V", OtherField: "VOther"},
		}
		kinds := func(m *suite.Method) []string {
			out := make([]string, 0, 4)
			for _, sm := range satMutantsOf(b, m) {
				out = append(out, sm.Kind)
			}
			return out
		}

		reader := stamp(projected("Get",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(namedRef("Value")), errRet}), "", "", "Value")
		testkit.Equal(t, kinds(reader),
			[]string{"inert", "flicker", "sputter", "spill", "flap"},
			"a pool-typed reader spills and flaps beside the shared kinds")

		scalar := projected("Count", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), errRet})
		testkit.Equal(t, kinds(scalar),
			[]string{"inert", "flicker", "sputter", "spill", "wane", "wax"},
			"an integer scalar wanes and waxes")

		replay := projected("Replay", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(sliceRef(namedRef("Entry"))), errRet})
		// fade alternates; the other three do not. A claim read off a single
		// drain is answered by whichever call the law makes, and the prover
		// isolates one law at a time — so an every-second defect lands on the
		// parity the law never sees.
		testkit.Equal(t, kinds(replay),
			[]string{"inert", "flicker", "sputter", "fade", "jumble", "dupdrain", "flood"},
			"a slice reader fades, jumbles, repeats and floods")

		page := projected("Page",
			[]golang.Param{arg("ctx", ctxRef()), arg("cur", namedRef("Cursor"))},
			[]golang.Return{
				res(sliceRef(namedRef("Value"))), res(namedRef("Cursor")), res(namedRef("bool")), errRet,
			})
		testkit.Equal(t, kinds(page), []string{"inert", "flicker", "sputter", "echo"},
			"a page-shaped walk echoes")

		// An operation reporting only an error carries the pair no
		// alternating defect can express: one that refuses after its first
		// call, and one that silently drops after it. An idempotence law
		// calls twice and discards the first answer, so a defect on every
		// other call is absorbed by the call nobody reads.
		op := projected("Close", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{errRet})
		testkit.Equal(t, kinds(op),
			[]string{"inert", "flicker", "sputter", "stick", "latch"},
			"an error-only operation sticks and latches")

		// A method taking a computation promises how often it runs it, and
		// only invoking that computation can break the promise.
		compute := projected("Run",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("fn", funcRef())},
			[]golang.Return{res(namedRef(qStr)), errRet})
		testkit.Assert(t, kinds(compute)).Contains("greedy",
			"a nullary computation parameter earns the extra invocation")

		// One that takes arguments does not: calling it would mean inventing
		// its inputs here, and a defect supplying its own inputs is testing
		// something else.
		fed := funcRef()
		fed.FuncParams = []*node.TypeRef{namedRef(qStr)}
		callback := projected("Each",
			[]golang.Param{arg("ctx", ctxRef()), arg("fn", fed)},
			[]golang.Return{errRet})
		for _, k := range kinds(callback) {
			testkit.NotEqual(t, k, "greedy", "a callback taking arguments is left alone")
		}

		// Every shape above carries flicker, and that is the point: a claim
		// about two calls agreeing is broken by an answer that changes, and
		// no shape-specific kind supplies one. Only a method answering
		// nothing has no flicker to wear.
		void := projected("Ping", []golang.Param{arg("ctx", ctxRef())}, nil)
		testkit.Equal(t, kinds(void), []string{"inert"},
			"a method answering nothing has nothing to flicker")
	})

	t.Run("a streamed result answers empty rather than nil", func(t *testing.T) {
		t.Parallel()
		// A stream's zero value is a nil function, and ranging over one
		// panics — so a wear answering the zero takes the run down before
		// the law it was worn for is asked anything.
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		seq2 := projected("List", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("iter", "Seq2"))})
		for _, sm := range satMutantsOf(b, seq2) {
			testkit.Equal(t, sm.Seq, 2, "the arity rides every wear on the method")
			testkit.Equal(t, sm.SeqHelper(), "EmptySeq2", "and names its helper")
		}

		seq1 := projected("Each", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("iter", "Seq"))})
		testkit.Equal(t, satMutantsOf(b, seq1)[0].SeqHelper(), "EmptySeq",
			"the one-value form names its own")

		// Each stream defect names the runtime helper that wears it, at the
		// arity the signature declares. A wrong name here is a template that
		// renders a call to a function that does not exist, which the corpus
		// catches — but only after a regeneration, and only for the arities
		// the corpus happens to hold.
		defects := map[string]string{}
		for _, sm := range satMutantsOf(b, seq2) {
			defects[sm.Kind] = sm.SeqDefect()
		}
		testkit.Equal(t, defects[kindFadeSeq], "FadedSeq2", "the faded drain")
		testkit.Equal(t, defects[kindDupSeq], "DoubledSeq2", "the doubled one")
		testkit.Equal(t, defects[kindFlood], "FloodedSeq2", "and the one that will not end")
		testkit.Equal(t, satMutantsOf(b, seq1)[0].SeqDefect(), "FadedSeq",
			"the one-value arity spells its own")

		plain := projected("Get", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("Value")), errRet})
		testkit.Equal(t, satMutantsOf(b, plain)[0].SeqHelper(), "",
			"a result that is not a stream names no helper")

		// The near misses, because the check is a name match against the
		// standard library and a wrong one would dress a stream defect on a
		// method that cannot carry it.
		notIter := projected("Chan", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("example.com/x", "Seq2"))})
		testkit.Equal(t, seqArity(notIter), 0, "Seq2 from another package is not one")
		wrongName := projected("Iter", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("iter", "Pull"))})
		testkit.Equal(t, seqArity(wrongName), 0, "iter has more than the two sequence types")
		twoResults := projected("Both", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(pkgRef("iter", "Seq")), errRet})
		testkit.Equal(t, seqArity(twoResults), 0,
			"a sequence beside an error is not the shape the drains take")
	})

	t.Run("the boundary wear needs a bound to cross", func(t *testing.T) {
		t.Parallel()
		// Built from the law's own stamped constant, so a law without one —
		// or with one no integer can be read from — has no line to step over
		// and earns no wear rather than a wear that steps nowhere.
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		count := projected("Count", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), errRet})
		h := harnessOf(count)

		noBound := &LawBinding{
			ID:     lawid.CountEqualsReference,
			Fields: []*LawField{{Name: "Count", Method: "Count"}},
		}
		_, ok := overshootOf(b, h, noBound, "Count")
		testkit.False(t, ok, "no stamped bound, no boundary to cross")

		fractional := &LawBinding{ID: lawid.AggregatorBounded, Fields: []*LawField{
			{Name: "Count", Method: "Count"}, {Name: fieldMax, Lit: "1.5"},
		}}
		_, ok = overshootOf(b, h, fractional, "Count")
		testkit.False(t, ok, "no counting shape answers one past a fraction")

		unread := &LawBinding{
			ID:     lawid.AggregatorBounded,
			Fields: []*LawField{{Name: fieldMax, Lit: "5"}},
		}
		_, ok = overshootOf(b, h, unread, "Count")
		testkit.False(t, ok, "a bound on a method the law does not read is not this law's line")
	})

	t.Run("the surface knows its reach and its restatement", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		b.Session = &SessionSpec{Reader: "Get", Writer: "Store"}
		b.Laws = []*LawBinding{
			{ID: lawid.MonotonicReads, Session: true},
			{ID: lawid.CountEqualsReference, Fields: []*LawField{{Name: "Count", Method: "Count"}}},
			{ID: lawid.PoolBalanced},
		}
		get := projected("Get", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(namedRef("Value")), errRet})
		store := projected("Store", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
			[]golang.Return{errRet})
		count := projected("Count", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), errRet})
		saturationOf(b, harnessOf(get, store, count))

		testkit.Equal(t, b.SatLaws[0].Methods, []string{"Get", "Store"},
			"a trace law wears defects on the session pair")
		testkit.True(t, b.SatLaws[1].AcceptSemantic,
			"the differential restated accepts the differential's own divergence")
		testkit.True(t, b.SatLaws[2].Unwearable,
			"a door-only law is skipped by name, never doomed")
	})
}

// A witnessed interface emits no prover: its wrappers would need the
// witness instantiation the surface does not thread.
func TestSaturationSkipsWitnessedInterfaces(t *testing.T) {
	t.Parallel()

	b := &Bindings{
		Subject:   suite.Subject{IfaceName: "Store"},
		Witnesses: []sdk.Ref{sdk.Builtin(qStr)},
		Laws:      []*LawBinding{{ID: lawid.ReadAfterWrite}},
	}
	saturationOf(b, harnessOf())
	testkit.Len(t, b.SatLaws, 0, "no prover over a generic surface")
}

// A law naming a method the projection does not carry wears nothing there —
// the prover's wardrobe stays honest about what it can dress.
func TestSaturationSkipsUnprojectedMethods(t *testing.T) {
	t.Parallel()

	b := &Bindings{
		Subject: suite.Subject{IfaceName: "Mixed"},
		Laws: []*LawBinding{{
			ID:     lawid.ReadAfterWrite,
			Fields: []*LawField{{Name: "Read", Method: "Nonesuch"}},
		}},
	}
	saturationOf(b, harnessOf())
	testkit.Equal(t, b.SatLaws[0].Methods, []string{"Nonesuch"},
		"the law still names its reach")
	testkit.Len(t, b.SatMutants, 0, "and nothing unprojected is dressed")
}

// TestLawSubsumption pins the dedup the optional roles forced: the same law
// re-selected from a partner carrier binds without the refinement only the
// directive's host resolves, and the richer binding must subsume the poorer
// in either arrival order — while genuinely distinct same-ID bindings, one
// per method, both stay.
func TestLawSubsumption(t *testing.T) {
	t.Parallel()

	rich := &LawBinding{ID: "AUTO-X", Fields: []*LawField{
		{Name: "Publish", Method: "Publish"},
		{Name: "Redeliver", Method: "Republish"},
	}}
	poor := &LawBinding{ID: "AUTO-X", Fields: []*LawField{
		{Name: "Publish", Method: "Publish"},
	}}
	other := &LawBinding{ID: "AUTO-X", Fields: []*LawField{
		{Name: "Publish", Method: "Broadcast"},
	}}

	testkit.True(t, lawSubsumes(rich, poor), "the refinement covers its own omission")
	testkit.False(t, lawSubsumes(poor, rich), "and never the other way around")
	testkit.False(t, lawSubsumes(rich, other), "a different resolved method is a different law instance")
	testkit.False(t, lawSubsumes(&LawBinding{ID: "AUTO-Y"}, poor), "as is a different identifier")
}
