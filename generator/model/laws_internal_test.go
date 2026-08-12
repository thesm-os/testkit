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
	fieldKey  = "Key"
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
			"trace-classifier": "eidos#25",
			"clock":            "aging-reference clock",
			"history":          "history hook",
			"coalesce-probe":   "does not construct",
		} {
			_, reason := handle(b, lawid.SingleflightCoalesces, "X", from, nil)
			testkit.Assert(t, reason).Contains(needle, from+" names its debt")
		}
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
