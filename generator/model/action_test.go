// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/gentest"
	"go.thesmos.sh/testkit/generator/model"
)

// TestActionSentinel pins the identity arm of the action comparison: a
// declaration that stamps a miss sentinel arms it on every error-answering
// reader, and one that stamps nothing leaves the comparison presence-only.
func TestActionSentinel(t *testing.T) {
	t.Parallel()

	readerOf := func(t *testing.T, b *model.Bindings) *model.Action {
		t.Helper()
		for _, a := range b.Actions {
			if a.Method == "Read" {
				return a
			}
		}
		t.Fatal("the fixture's reader was not driven")
		return nil
	}

	t.Run("a stamped sentinel rides the reader", func(t *testing.T) {
		t.Parallel()
		s := mixed(t)
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name != "Read" {
					continue
				}
				shape.MetaMixins.Set(m.EnsureMeta(), []string{"notfound"}, "test")
				shape.MixinParamKey("notfound", "sentinel").
					Set(m.EnsureMeta(), "example.com/validates.ErrGone", "test")
			}
		}
		b := bindingsOf(t, s)
		testkit.True(t, readerOf(t, b).Sentinel != nil,
			"the declaration's miss identity reaches the sequences, not only the oracle")
	})

	t.Run("an unstamped declaration stays presence-only", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, mixed(t))
		testkit.True(t, readerOf(t, b).Sentinel == nil,
			"no stamp, no identity to hold the pair to")
	})
}

// TestActionShapes walks every detector arm the action derivation covers: one
// method per shape, each asserted to its template kind, its pools, and its
// drawn types — plus the shapes that skip, each with the reason the header
// prints. The store is never rendered, so the types need only be consistent
// enough for the derivation to read.
func TestActionShapes(t *testing.T) {
	t.Parallel()

	str := func() *sdk.TypeRef { return storefixture.Named("string") }
	ch := storefixture.Named("string")
	golang.MetaIsChannel.Set(ch.EnsureMeta(), true, "test")

	s := storefixture.New().
		Package("zoo", "example.com/zoo").
		Interface("Zoo", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("zoo/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			method := func(name string, params, returns []*sdk.TypeRef) {
				i.Method(name, func(m *storefixture.MethodBuilder) {
					gentest.Ctx(m)
					for n, p := range params {
						m.Param(string(rune('a'+n)), p)
					}
					for _, r := range returns {
						m.Return(r)
					}
				})
			}
			method("Get", []*sdk.TypeRef{str()}, []*sdk.TypeRef{str(), storefixture.Named("error")})
			method("Put", []*sdk.TypeRef{str()}, []*sdk.TypeRef{storefixture.Named("error")})
			method("NoErr", []*sdk.TypeRef{str()}, []*sdk.TypeRef{str()})
			method("Find", []*sdk.TypeRef{str()}, []*sdk.TypeRef{str()})
			method("Load", []*sdk.TypeRef{str()}, []*sdk.TypeRef{str(), storefixture.Named("bool")})
			method("Meta", []*sdk.TypeRef{str()}, []*sdk.TypeRef{str(), str(), storefixture.Named("error")})
			method("GetAll", []*sdk.TypeRef{str()},
				[]*sdk.TypeRef{storefixture.Slice(str()), storefixture.Named("error")})
			method("Touch", []*sdk.TypeRef{str()}, nil)
			method("Stats", nil, []*sdk.TypeRef{str(), str(), storefixture.Named("error")})
			method("Size", nil, []*sdk.TypeRef{storefixture.Named("int")})
			method("Iter", nil, []*sdk.TypeRef{str()})
			method("BadIter", nil, []*sdk.TypeRef{str()})
			method("Ingest", []*sdk.TypeRef{str()}, []*sdk.TypeRef{storefixture.Named("error")})
			method("Stop", nil, nil)
			method("Watch", []*sdk.TypeRef{str()}, []*sdk.TypeRef{ch, storefixture.Named("error")})
			i.Method("Inspect", func(m *storefixture.MethodBuilder) {
				m.Param("k", str())
				m.Return(str())
				m.Return(str())
				m.Return(storefixture.Named("bool"))
			})
			i.Method("Err", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Build()

	for name, shapeName := range map[string]string{
		"Get": "reader", "Put": "writer", "NoErr": "readernoerror",
		"Find": "pointerreader", "Load": "readerwithbool", "Meta": "multireader",
		"GetAll": "batchreader", "Touch": "mutator", "Stats": "multiaggregator",
		"Size": "aggregator", "Iter": "streamreader", "BadIter": "streamreader",
		"Ingest": "streamconsumer", "Stop": "voidlifecycle", "Watch": "reader",
		"Inspect": "lookup", "Err": "poisonaccessor",
	} {
		stampShape(s, name, shapeName, "string", "string")
	}
	// BadIter deliberately loses its value stamp: the drain has nothing to
	// spell its elements with.
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == "BadIter" {
				shape.MetaValueType.Set(m.EnsureMeta(), "", "test")
			}
		}
	}

	b := bindingsOf(t, s)
	kinds := map[string]sdk.Kind{}
	pools := map[string]string{}
	byName := map[string]*model.Action{}
	for _, a := range b.Actions {
		kinds[a.Method] = a.KindName
		pools[a.Method] = a.Pool
		byName[a.Method] = a
	}
	skips := map[string]string{}
	for _, sk := range b.Skipped {
		skips[sk.Method] = sk.Reason
	}

	testkit.Equal(t, kinds["NoErr"], sdk.Kind("model.action.readernoerror"), "the errorless reader")
	testkit.Equal(t, kinds["Find"], sdk.Kind("model.action.pointerreader"), "the pointer reader")
	testkit.Equal(t, kinds["Load"], sdk.Kind("model.action.readerwithbool"), "the found-flag reader")
	testkit.Equal(t, kinds["Meta"], sdk.Kind("model.action.multireader"), "the two-value reader")
	testkit.Equal(t, kinds["GetAll"], sdk.Kind("model.action.batchreader"), "the batch reader")
	testkit.Equal(t, kinds["Touch"], sdk.Kind("model.action.mutator"), "the mutator")
	testkit.Equal(t, kinds["Stats"], sdk.Kind("model.action.multiaggregator"), "the two-value aggregator")
	testkit.Equal(t, kinds["Iter"], sdk.Kind("model.action.streamreader"), "the iterator drain")
	testkit.Equal(t, kinds["Stop"], sdk.Kind("model.action.voidlifecycle"), "the void lifecycle")
	testkit.Equal(t, kinds["Inspect"], sdk.Kind("model.action.lookup"), "the context-free lookup")
	testkit.Equal(t, kinds["Err"], sdk.Kind("model.action.poisonaccessor"), "the poison accessor")

	testkit.Equal(t, pools["Meta"], "keys", "a multi-reader draws keys")
	testkit.Equal(t, pools["Touch"], "values", "a mutator draws values")
	testkit.True(t, byName["Meta"].Value2 != nil, "the second value is spelled")
	testkit.True(t, byName["Size"].NoError, "an errorless aggregator supplies nil itself")
	testkit.True(t, byName["GetAll"].Value != nil, "the batch's element is the slice's")

	testkit.Assert(t, skips["Ingest"]).Contains("caller-built stream",
		"the consumer's stream is nothing a derivation constructs")
	testkit.Assert(t, skips["BadIter"]).Contains("no stamp",
		"an unstamped iterator has nothing to spell its elements with")
	testkit.Assert(t, skips["Watch"]).Contains("live handle",
		"a channel compares by identity, which two sides never share")
}

// TestInertActionsAreSkipped pins the coherence rule between the sequences
// and the adapter: an action on a method the derived reference holds inert
// compares the subject against a body answering zeros, and is skipped with
// the adapter's own reason instead. The corpus proved the rule — a keyed
// oracle has no drain, and the first driven drain diverged from the inert
// body's nil at the first held key.
func TestInertActionsAreSkipped(t *testing.T) {
	t.Parallel()

	s := keyedStoreWith(t, "example.com/kv.ErrGone", func(i *storefixture.InterfaceBuilder) {
		i.Method("List", func(m *storefixture.MethodBuilder) {
			gentest.Ctx(m)
			m.Return(storefixture.Slice(storefixture.Named("string")))
			gentest.Err(m)
		})
	})
	stampShape(s, "List", "aggregator", "", "")
	b := bindingsOf(t, s)
	testkit.Equal(t, b.Reference.StoreType(), "KeyedStore", "the pair still derives the keyed oracle")
	reasons := map[string]string{}
	for _, sk := range b.Skipped {
		reasons[sk.Method] = sk.Reason
	}
	testkit.Assert(t, reasons["List"]).Contains("holds it inert",
		"the drain the oracle cannot answer is skipped, not driven against zeros")
	for _, a := range b.Actions {
		testkit.NotEqual(t, a.Method, "List", "and never appears in the sequences")
	}
}

// TestParameterisedPureDrawsItsArguments pins the purevar refinement: a pure
// method with inputs drives through the drawn-args constructor, each position
// wide where its type can be seen to the bottom.
func TestParameterisedPureDrawsItsArguments(t *testing.T) {
	t.Parallel()

	// Pure alone rides the twin floor: on a derived store the oracle holds a
	// pure method inert and the coherence rule skips its action, so the
	// drawn-args path is the twin's to prove.
	s := storefixture.New().
		Package("fn", "example.com/fn").
		Interface("Deriver", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("fn/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			i.Method("Derive", func(m *storefixture.MethodBuilder) {
				m.Param("input", storefixture.Named("string"))
				m.Return(storefixture.Named("string"))
			})
		}).
		Build()
	stampShape(s, "Derive", "pure", "", "string")
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Twin(), "nothing here derives a store")

	var derive *model.Action
	for _, a := range b.Actions {
		if a.Method == "Derive" {
			derive = a
		}
	}
	testkit.True(t, derive != nil, "the parameterised pure call is driven")
	testkit.Equal(t, derive.KindName, sdk.Kind("model.action.purevar"),
		"through the drawn-args template")
	testkit.Equal(t, len(derive.Args), 1, "one drawn position")
	testkit.True(t, derive.Args[0].Wide,
		"wide, because a pure call stores nothing a claim could refuse")
}
