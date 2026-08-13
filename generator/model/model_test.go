// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/model"
	"go.thesmos.sh/testkit/generator/suite"
)

// TestConformance holds the plugin to the pipeline's own contract.
func TestConformance(t *testing.T) {
	t.Parallel()
	plugintest.RunSuite(t, model.New())
}

// TestBindings walks the derivation for the corpus's own fixture shape: a
// stamped writer carrying the validates mixin, its partner, and a reader.
func TestBindings(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, mixed(t))

	t.Run("drives the reader and the writer", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, len(b.Actions), 2, "two methods map to actions")
		testkit.Equal(t, b.Actions[0].Method, "Store", "in declaration order")
		testkit.Equal(t, b.Actions[0].KindName, sdk.Kind("model.action.writer"),
			"the writer renders through its shape's template")
		testkit.Equal(t, b.Actions[1].Method, "Read", "the reader follows")
	})

	t.Run("excludes the partner with its reason", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, len(b.Skipped), 1, "one method is not driven")
		testkit.Equal(t, b.Skipped[0].Method, "Validate",
			"the method the mixin references")
		testkit.Assert(t, b.Skipped[0].Reason).Contains("validates.fn",
			"naming the stamp that claims it")
	})

	t.Run("derives the map reference", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, b.Reference.Supplied(), "nothing was named, so it is derived")
		testkit.Equal(t, b.Reference.KeyField, "Key",
			"keyed on the one string field matching the reader's key")
		testkit.Equal(t, len(b.Adapter), 3, "every method has an adapter body")
		testkit.Equal(t, b.Adapter[0].Op, "Put", "the writer delegates")
		testkit.Equal(t, b.Adapter[2].Op, "Get", "the reader delegates")
		testkit.True(t, b.Adapter[1].Op == "" && b.Adapter[1].Reason != "",
			"the partner is inert, with why")
	})

	t.Run("draws from the fixture pools", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, b.Keys.Field, "Key", "keys are the reader's fixture pair")
		testkit.Equal(t, b.Keys.OtherField, "KeyOther", "with the companion beside it")
		testkit.Equal(t, b.Values.Field, "V", "values are the writer's")
	})

	t.Run("the validates claim narrows the values", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, b.Values.Wide, "a validating subject may refuse a raw draw")
		testkit.Assert(t, b.Values.WhyNarrow).Contains("validates claim on Store",
			"the header names the claim and its carrier")
		testkit.Equal(t, b.Values.Pin, "",
			"and even a recombined fixture body is a value nothing proved accepted")
	})
}

// TestSuppliedReference pins the escape hatch: ref= replaces the derivation
// wholesale, so nothing else needs to be derivable.
func TestSuppliedReference(t *testing.T) {
	t.Parallel()

	s := mixed(t, storefixture.KV(model.RefKey, "NewFake"))
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Supplied(), "the directive named the constructor")
	testkit.Equal(t, len(b.Adapter), 0, "and no adapter is generated over it")

	t.Run("an empty value is no supply", func(t *testing.T) {
		t.Parallel()
		// `ref=` with nothing after it reads as a slip, not a choice; the
		// derivation proceeds as though the key were absent.
		b := bindingsOf(t, mixed(t, storefixture.KV(model.RefKey, "")))
		testkit.False(t, b.Reference.Supplied(), "the empty value fell through to derivation")
	})

	t.Run("a reader alone draws no values", func(t *testing.T) {
		t.Parallel()
		// The pool declarations are conditional on use because an unused local
		// is a compile error in every generated file. ref= is what makes a
		// one-sided interface reachable at all.
		s := storefixture.New().
			Package("ro", "example.com/ro").
			Interface("Fetcher", func(i *storefixture.InterfaceBuilder) {
				i.Pos(sdk.At("ro/iface.go", 1, 1))
				i.Directive(storefixture.Directive("suite"))
				i.Directive(storefixture.Directive("model",
					storefixture.KV(model.RefKey, "NewFake")))
				i.Method("Fetch", func(m *storefixture.MethodBuilder) {
					m.Param("ctx", storefixture.PkgNamed("context", "Context"))
					m.Param("key", storefixture.Named("string"))
					m.Return(storefixture.Named("string"))
					m.Return(storefixture.Named("error"))
				})
			}).
			Build()
		stampShape(s, "Fetch", "reader", "string", "string")
		b := bindingsOf(t, s)
		testkit.True(t, b.UsesKeys(), "the reader draws keys")
		testkit.False(t, b.UsesValues(), "and nothing draws values")
	})

	t.Run("a writer alone draws no keys", func(t *testing.T) {
		t.Parallel()
		s := storefixture.New().
			Package("wo", "example.com/wo").
			Interface("Sink", func(i *storefixture.InterfaceBuilder) {
				i.Pos(sdk.At("wo/iface.go", 1, 1))
				i.Directive(storefixture.Directive("suite"))
				i.Directive(storefixture.Directive("model",
					storefixture.KV(model.RefKey, "NewFake")))
				i.Method("Push", func(m *storefixture.MethodBuilder) {
					m.Param("ctx", storefixture.PkgNamed("context", "Context"))
					m.Param("v", storefixture.Named("string"))
					m.Return(storefixture.Named("error"))
				})
			}).
			Build()
		stampShape(s, "Push", "writer", "", "string")
		b := bindingsOf(t, s)
		testkit.True(t, b.UsesValues(), "the writer draws values")
		testkit.False(t, b.UsesKeys(), "and nothing draws keys")
	})
}

// TestRenderSurface hits what only the templates otherwise read, so a rename
// that breaks the template's field lookup fails here with a name rather than
// in the backend with a short file.
func TestRenderSurface(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, mixed(t))
	testkit.Equal(t, b.Kind(), model.KindBindings, "the bindings render as themselves")
	testkit.Equal(t, b.Actions[0].Kind(), b.Actions[0].KindName,
		"an action renders through its shape's template")
	testkit.Equal(t, b.ModelPkg(), model.ModelPkg, "the runner's import path")
	testkit.Equal(t, b.RefPkg(), model.RefPkg, "the oracle's import path")
	testkit.Equal(t, b.TierName(), model.TierName, "the path the run reports under")
	testkit.True(t, b.UsesKeys() && b.UsesValues(),
		"a reader and a writer draw from both pools")

	t.Run("the miss prefix follows the routed package", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, b.MissPrefix(), "mixed",
			"provisional before Layout: the interface's own spelling")
		b := bindingsOf(t, mixed(t))
		b.SetOutputPackages(map[string]string{"": "example.com/validates/validatestest"})
		testkit.Equal(t, b.MissPrefix(), "validatestest",
			"and the routed package once Layout resolved it")
		b.SetOutputPackages(map[string]string{})
		testkit.Equal(t, b.MissPrefix(), "validatestest",
			"a partial map on a later call clears nothing")
	})
}

// TestUnmappableMethods holds the skip arms to their reasons: an unstamped
// method, one stamped outside the table, and one that cannot forward to the
// oracle are all listed — never silently absent from the sequences.
func TestUnmappableMethods(t *testing.T) {
	t.Parallel()

	s := mixedWith(t, func(i *storefixture.InterfaceBuilder) {
		// Unstamped: the annotator classified nothing.
		i.Method("Ping", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Return(storefixture.Named("error"))
		})
		// Stamped with vocabulary this build's table has never heard of — the
		// state a detector landing upstream is in until the row lands.
		i.Method("Weird", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Return(storefixture.Named("error"))
		})
		// Stamped and driven, but with no context to forward to the oracle.
		i.Method("Len", func(m *storefixture.MethodBuilder) {
			m.Return(storefixture.Named("int"))
			m.Return(storefixture.Named("error"))
		})
	})
	stampShape(s, "Weird", "not-a-shape", "", "")
	stampShape(s, "Len", "aggregator", "", "int")
	// A mixin the registry has never heard of exercises the partner scan's
	// miss arm without disturbing the real one.
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == "Ping" {
				shape.MetaMixins.Set(m.EnsureMeta(), []string{"not-a-mixin"}, "test")
			}
		}
	}
	b := bindingsOf(t, s)

	reasons := map[string]string{}
	for _, sk := range b.Skipped {
		reasons[sk.Method] = sk.Reason
	}
	testkit.Assert(t, reasons["Ping"]).Contains("no shape",
		"the unstamped method names the classification gap")
	testkit.Assert(t, reasons["Weird"]).Contains("not-a-shape",
		"the unmapped stamp names the shape nothing drives")

	inert := map[string]string{}
	for _, am := range b.Adapter {
		if am.Op == "" {
			inert[am.Sig.Name] = am.Reason
		}
	}
	testkit.Assert(t, inert["Len"]).Contains("no context",
		"a driven method the oracle cannot serve is inert, with why")
	testkit.Assert(t, inert["Ping"]).Contains("shape",
		"an unclassified method is inert too")
}

// TestDiagnostics holds every refusal to a message naming what fixes it.
func TestDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("an interface with no harness", func(t *testing.T) {
		t.Parallel()
		s := storefixture.New().
			Package("bare", "example.com/bare").
			Interface("Bare", func(i *storefixture.InterfaceBuilder) {
				i.Pos(sdk.At("bare/iface.go", 1, 1))
				i.Directive(storefixture.Directive("model"))
				i.Method("Get", func(m *storefixture.MethodBuilder) {
					m.Param("ctx", storefixture.PkgNamed("context", "Context"))
					m.Param("key", storefixture.Named("string"))
					m.Return(storefixture.Named("string"))
					m.Return(storefixture.Named("error"))
				})
			}).
			Build()
		got := plugintest.Generate(t, model.New(), s).Diagnostics()
		testkit.Equal(t, len(got), 1, "one diagnostic")
		testkit.Assert(t, got[0].Message).Contains("//testkit:suite",
			"naming the directive that creates the projection this binds onto")
	})

	t.Run("a generic interface", func(t *testing.T) {
		t.Parallel()
		s := storefixture.New().
			Package("gen", "example.com/gen").
			Interface("Store", func(i *storefixture.InterfaceBuilder) {
				i.Pos(sdk.At("gen/iface.go", 1, 1))
				i.Directive(storefixture.Directive("suite"))
				i.Directive(storefixture.Directive("model"))
				i.TypeParam("V", nil)
				i.Method("Put", func(m *storefixture.MethodBuilder) {
					m.Param("ctx", storefixture.PkgNamed("context", "Context"))
					m.Param("v", storefixture.TypeParamRef("V"))
					m.Return(storefixture.Named("error"))
				})
			}).
			Build()
		plugintest.Generate(t, suite.New(), s)
		got := plugintest.Generate(t, model.New(), s).Diagnostics()
		testkit.Equal(t, len(got), 1, "one diagnostic")
		testkit.Assert(t, got[0].Message).Contains("witness=",
			"the property and the reference land at concrete types, and the key names them")
	})

	t.Run("a witness list disagreeing with the parameter list", func(t *testing.T) {
		t.Parallel()
		s := genericFixture(t, storefixture.KV(model.WitnessKey, "string,int"))
		plugintest.Generate(t, suite.New(), s)
		got := plugintest.Generate(t, model.New(), s).Diagnostics()
		testkit.Equal(t, len(got), 1, "one diagnostic")
		testkit.Assert(t, got[0].Message).Contains("one per parameter",
			"a partial list would leave the generator guessing which position it means")
	})

	t.Run("a witnessed generic interface emits at the witnesses", func(t *testing.T) {
		t.Parallel()
		s := genericFixture(t, storefixture.KV(model.WitnessKey, "int"))
		plugintest.Generate(t, suite.New(), s)
		res := plugintest.Generate(t, model.New(), s)
		testkit.Equal(t, len(res.Diagnostics()), 0, "the witness answers the refusal")
		b := bindingsOf(t, s)
		testkit.Equal(t, len(b.Witnesses), 1, "and the emission carries it")
	})

	t.Run("a qualified ref constructor", func(t *testing.T) {
		t.Parallel()
		s := mixed(t, storefixture.KV(model.RefKey, "other.NewFake"))
		got := generateBoth(t, s).Diagnostics()
		testkit.Equal(t, len(got), 1, "one diagnostic")
		testkit.Assert(t, got[0].Message).Contains("own package",
			"the generator cannot invent an import path for a foreign qualifier")
	})

	t.Run("no method maps to an action", func(t *testing.T) {
		t.Parallel()
		s := storefixture.New().
			Package("un", "example.com/un").
			Interface("Opaque", func(i *storefixture.InterfaceBuilder) {
				i.Pos(sdk.At("un/iface.go", 1, 1))
				i.Directive(storefixture.Directive("suite"))
				i.Directive(storefixture.Directive("model"))
				i.Method("Do", func(m *storefixture.MethodBuilder) {
					m.Param("ctx", storefixture.PkgNamed("context", "Context"))
					m.Return(storefixture.Named("error"))
				})
			}).
			Build()
		got := generateBoth(t, s).Diagnostics()
		testkit.Equal(t, len(got), 1, "one diagnostic")
		testkit.Assert(t, got[0].Message).Contains("drive",
			"sequences over nothing assert nothing")
	})
}

// TestTwinFloor walks the fallback that arms what no store models: the
// reference becomes the subject's own factory, the header carries why, and
// nothing derived — no adapter, no companion — rides along.
func TestTwinFloor(t *testing.T) {
	t.Parallel()

	twinOf := func(t *testing.T, s *sdk.Store) *model.Bindings {
		t.Helper()
		b := bindingsOf(t, s)
		testkit.True(t, b.Reference.Twin(), "the floor is the twin")
		testkit.False(t, b.Reference.Derived(), "which is not a derivation")
		testkit.Equal(t, len(b.Adapter), 0, "so no adapter is generated")
		return b
	}

	t.Run("a reader alone", func(t *testing.T) {
		t.Parallel()
		b := twinOf(t, readerOnly(t))
		testkit.Assert(t, b.Reference.TwinWhy).Contains("no reader/writer pair",
			"the header says what was missing")
	})

	t.Run("reader and writer disagree about the value", func(t *testing.T) {
		t.Parallel()
		b := twinOf(t, kvStore(t, "example.com/kv.Doc", "string"))
		testkit.Assert(t, b.Reference.TwinWhy).Contains("where the writer takes",
			"two value types share no store, but twins compare anything")
	})

	t.Run("the value declares no struct to key on", func(t *testing.T) {
		t.Parallel()
		b := twinOf(t, kvStore(t, "string", "string"))
		testkit.Assert(t, b.Reference.TwinWhy).Contains("no struct declaration",
			"naming what the key projection needed")
	})

	t.Run("no field carries the key type", func(t *testing.T) {
		t.Parallel()
		b := twinOf(t, kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
			field{"N", "int"}))
		testkit.Assert(t, b.Reference.TwinWhy).Contains("no field",
			"nothing to project a key out of")
	})

	t.Run("several fields tie without a conventional name", func(t *testing.T) {
		t.Parallel()
		b := twinOf(t, kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
			field{"A", "string"}, field{"B", "string"}))
		testkit.Assert(t, b.Reference.TwinWhy).Contains("several fields",
			"an arbitrary pick would key a real oracle on a guess")
		testkit.True(t, b.Values.Wide, "and the twin stays wide unpinned — twins agree on misses")
		testkit.Equal(t, b.Values.Pin, "", "with no field to pin")
	})

	t.Run("a second writer of another type is skipped by name", func(t *testing.T) {
		t.Parallel()
		s := mixedWith(t, func(i *storefixture.InterfaceBuilder) {
			i.Method("StoreRaw", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("v", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
		})
		stampShape(s, "StoreRaw", "writer", "", "string")
		b := bindingsOf(t, s)
		testkit.Equal(t, b.Reference.StoreType(), "MapStore",
			"the matching writer still derives the map")
		reasons := map[string]string{}
		for _, sk := range b.Skipped {
			reasons[sk.Method] = sk.Reason
		}
		testkit.Assert(t, reasons["StoreRaw"]).Contains("values pool draws",
			"the odd writer is listed, not miscompiled")
	})
}

// TestKeyedReference walks the keyed-put derivation: a composite writer
// selects the keyed store, needs no projection, and the delete's mixin — not
// its shape — names its oracle operation.
func TestKeyedReference(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, keyedStore(t, "example.com/kv.ErrGone"))

	t.Run("the composite writer selects the keyed oracle", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, b.Reference.Keyed(), "the key is an argument, not a projection")
		testkit.Equal(t, b.Reference.StoreType(), "KeyedStore", "and the store is the keyed one")
		testkit.Equal(t, b.Reference.KeyField, "", "with no field to project")
	})

	t.Run("the delete delegates by its mixin, not its shape", func(t *testing.T) {
		t.Parallel()
		ops := map[string]string{}
		for _, am := range b.Adapter {
			ops[am.Sig.Name] = am.Op
		}
		testkit.Equal(t, ops["Del"], "Delete", "writer-shaped, delete-stamped")
		testkit.Equal(t, ops["Put"], "Put", "the composite put delegates as itself")
		testkit.Equal(t, ops["Get"], "Get", "and the reader reads")
	})

	t.Run("the delete draws keys, the put draws both", func(t *testing.T) {
		t.Parallel()
		pools := map[string]string{}
		for _, a := range b.Actions {
			pools[a.Method] = a.Pool
		}
		testkit.Equal(t, pools["Del"], "keys", "a delete's argument is a key")
		testkit.Equal(t, b.Values.Field, "Value", "values come from the put's second argument")
		testkit.True(t, b.UsesKeys() && b.UsesValues(), "both pools are declared")
	})

	t.Run("an identical second selection binds once", func(t *testing.T) {
		t.Parallel()
		// Two delete-stamped writers naming the same read partner select the
		// delete law twice with one field set; the registry must carry it
		// once, not report one failure twice.
		s := keyedStoreWith(t, "example.com/kv.ErrGone", func(i *storefixture.InterfaceBuilder) {
			i.Method("Del2", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
		})
		stampShape(s, "Del2", "writer", "string", "")
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name != "Del2" {
					continue
				}
				bag := m.EnsureMeta()
				shape.MetaMixins.Set(bag, []string{"deleteremoves"}, "test")
				shape.MixinParamKey("deleteremoves", "read").
					Set(bag, "example.com/kv.Get", "test")
				shape.MixinParamKey("deleteremoves", "sentinel").
					Set(bag, "example.com/kv.ErrGone", "test")
			}
		}
		got := bindingsOf(t, s)
		bound := 0
		for _, l := range got.Laws {
			if l.ID == lawid.DeleteReturnsNotFound {
				bound++
			}
		}
		testkit.Equal(t, bound, 1, "one binding, whichever carrier selected it")
	})

	t.Run("a partner stamp cleared refuses by name", func(t *testing.T) {
		t.Parallel()
		s := keyedStore(t, "example.com/kv.ErrGone")
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Del" {
					shape.MixinParamKey("deleteremoves", "read").Set(m.EnsureMeta(), "", "test")
				}
			}
		}
		got := bindingsOf(t, s)
		unbound := map[string]string{}
		for _, u := range got.Unbound {
			unbound[u.Method] = u.Reason
		}
		testkit.Assert(t, unbound[lawid.DeleteReturnsNotFound]).Contains("does not stamp",
			"the missing stamp is the named cause")
	})

	t.Run("a partner naming a non-method refuses by name", func(t *testing.T) {
		t.Parallel()
		s := keyedStore(t, "example.com/kv.ErrGone")
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Del" {
					shape.MixinParamKey("deleteremoves", "read").
						Set(m.EnsureMeta(), "example.com/kv.Nowhere", "test")
				}
			}
		}
		got := bindingsOf(t, s)
		unbound := map[string]string{}
		for _, u := range got.Unbound {
			unbound[u.Method] = u.Reason
		}
		testkit.Assert(t, unbound[lawid.DeleteReturnsNotFound]).Contains("not a method",
			"the dangling name is the named cause")
	})

	t.Run("a partner reader off the pools' types refuses by name", func(t *testing.T) {
		t.Parallel()
		// The delete's partner is a second reader answering int beside the
		// string pools, which no generated closure can reconcile — and being
		// the later declaration it also carries the pools, so the store
		// falls to the twin while the law refuses on the types.
		s := keyedStoreWith(t, "example.com/kv.ErrGone", func(i *storefixture.InterfaceBuilder) {
			i.Method("GetOther", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("int"))
				m.Return(storefixture.Named("error"))
			})
		})
		stampShape(s, "GetOther", "reader", "string", "int")
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Del" {
					shape.MixinParamKey("deleteremoves", "read").
						Set(m.EnsureMeta(), "example.com/kv.GetOther", "test")
				}
			}
		}
		got := bindingsOf(t, s)
		unbound := map[string]string{}
		for _, u := range got.Unbound {
			unbound[u.Method] = u.Reason
		}
		testkit.Assert(t, unbound[lawid.DeleteReturnsNotFound]).Contains("beside pools of",
			"the type disagreement is the named cause")
	})

	t.Run("the sentinel binds from the stamp", func(t *testing.T) {
		t.Parallel()
		var bound *model.LawBinding
		for _, l := range b.Laws {
			if l.ID == lawid.DeleteReturnsNotFound {
				bound = l
			}
		}
		testkit.True(t, bound != nil, "the delete law binds")
		testkit.Equal(t, bound.Fields[0].Method, "Get",
			"Read resolves through the deleteremoves.read partner stamp")
		testkit.True(t, bound.Fields[2].Const != nil,
			"Sentinel renders the stamped constant")
		testkit.Equal(t, bound.Kind(), sdk.Kind("model.law"),
			"a binding renders through the one law template")
		testkit.Equal(t, bound.Fields[0].ModelPkg(), model.ModelPkg,
			"and its closures reach the runner's package")
	})
}

// TestUnqualifiedSentinelIsRefused pins the constant renderer's refusal: a
// stamp without a package is a name the generator cannot import — bare or
// dangling, the two ways a qualifier can fail to be one.
func TestUnqualifiedSentinelIsRefused(t *testing.T) {
	t.Parallel()

	for _, sentinel := range []string{"ErrGone", "example.com/kv."} {
		b := bindingsOf(t, keyedStore(t, sentinel))
		unbound := map[string]string{}
		for _, u := range b.Unbound {
			unbound[u.Method] = u.Reason
		}
		testkit.Assert(t, unbound[lawid.DeleteReturnsNotFound]).Contains("neither a qualified symbol",
			"the refusal names what the stamp is missing")
	}
}

// TestKeyedMismatchFallsToTheTwin pins the keyed fork's agreement: the reader
// and the keyed writer speak one (K, V) pair, or no store models them and the
// twin floor stands in with the disagreement in the header.
func TestKeyedMismatchFallsToTheTwin(t *testing.T) {
	t.Parallel()

	s := keyedStore(t, "example.com/kv.ErrGone")
	stampShape(s, "Get", "reader", "int", "string")
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Twin(), "one store cannot model the pair")
	testkit.Assert(t, b.Reference.TwinWhy).Contains("keyed writer takes",
		"and the header spells the disagreement")
}

// keyedStore is the keyed-put fixture: a reader, a composite writer, and a
// delete-stamped plain writer, with the delete law's stamps carried.
func keyedStore(t *testing.T, sentinel string) *sdk.Store {
	t.Helper()
	return keyedStoreWith(t, sentinel, nil)
}

// keyedStoreWith is [keyedStore] plus extra methods.
func keyedStoreWith(t *testing.T, sentinel string, extra func(i *storefixture.InterfaceBuilder)) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("kv", "example.com/kv").
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("kv/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Param("value", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Del", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
			if extra != nil {
				extra(i)
			}
		}).
		Build()
	stampShape(s, "Get", "reader", "string", "string")
	stampShape(s, "Put", "compositewriter", "string", "string")
	stampShape(s, "Del", "writer", "string", "")
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Del" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{"deleteremoves"}, "test")
			shape.MixinParamKey("deleteremoves", "read").
				Set(bag, "example.com/kv.Get", "test")
			shape.MixinParamKey("deleteremoves", "sentinel").
				Set(bag, sentinel, "test")
		}
	}
	return s
}

// TestValuePoolWidth walks the widening decision's arms: the license the
// claims grant, the pin that keeps a wide draw colliding, and the two ways a
// pool stays narrow without a restricting claim.
func TestValuePoolWidth(t *testing.T) {
	t.Parallel()

	t.Run("a scalar-fielded value goes wide, keyed from the pool", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
			field{"ID", "string"}, field{"N", "int"}))
		testkit.True(t, b.Values.Wide, "nothing in the claims restricts the domain")
		testkit.Equal(t, b.Values.Pin, "ID", "and every draw lands on a pooled key")
		testkit.Equal(t, b.Values.WhyNarrow, "", "so there is nothing to explain")
	})

	t.Run("an unexported field is skipped the way Make skips it", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
			field{"ID", "string"}, field{"body", "string"}))
		testkit.True(t, b.Values.Wide, "Make leaves it zero, which draws fine")
	})

	t.Run("a field out of reach keeps the pair, pinned", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
			field{"ID", "string"}, field{"When", "time.Time"}))
		testkit.False(t, b.Values.Wide, "a wide draw would arm a run-time panic")
		testkit.Assert(t, b.Values.WhyNarrow).Contains("time.Time", "naming the reach")
		testkit.Equal(t, b.Values.Pin, "ID",
			"recombining proven bodies with pooled keys is still licensed")
	})

	t.Run("a keyed put widens with no pin", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, keyedStore(t, "example.com/kv.ErrGone"))
		testkit.True(t, b.Values.Wide, "a scalar value serves a wide draw")
		testkit.Equal(t, b.Values.Pin, "", "the key is an argument, not a field")
	})

	t.Run("a supplied reference retries the pin", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStoreWith(t, "example.com/kv.Doc", "example.com/kv.Doc",
			[]storefixture.DirectiveOption{storefixture.KV(model.RefKey, "NewFake")},
			field{"ID", "string"}))
		testkit.True(t, b.Reference.Supplied(), "ref= replaced the derivation")
		testkit.Equal(t, b.Values.Pin, "ID", "but the pin derives on its own")
		testkit.True(t, b.Values.Wide, "so the wide pool still lands on pooled keys")
	})

	t.Run("a supplied reference with no derivable pin narrows", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStoreWith(t, "example.com/kv.Doc", "example.com/kv.Doc",
			[]storefixture.DirectiveOption{storefixture.KV(model.RefKey, "NewFake")}))
		testkit.False(t, b.Values.Wide, "wide values keyed afresh never collide")
		testkit.Assert(t, b.Values.WhyNarrow).Contains("pin a wide draw",
			"and the header says what is missing")
	})
}

// TestStickyRefinement walks the conflict the corpus surfaced the day the
// pools went wide: a sticky reader refines the oracle to its pinning form,
// and negates the observability law the writer's shape would otherwise earn —
// on a sticky store the two claims contradict at the first overwrite.
func TestStickyRefinement(t *testing.T) {
	t.Parallel()

	s := kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc", field{"ID", "string"})
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == "Get" {
				shape.MetaMixins.Set(m.EnsureMeta(), []string{"sticky"}, "test")
			}
		}
	}
	b := bindingsOf(t, s)

	testkit.True(t, b.Reference.Pins, "the reader's claim refines the oracle")
	testkit.Equal(t, b.Reference.StoreType(), "StickyStore", "to its pinning form")

	unbound := map[string]string{}
	for _, u := range b.Unbound {
		unbound[u.Method] = u.Reason
	}
	testkit.Assert(t, unbound[lawid.WriteObservable]).Contains("sticky claim",
		"the negated law is listed with the contradiction, not silently absent")
	for _, l := range b.Laws {
		testkit.NotEqual(t, l.ID, lawid.WriteObservable, "and never bound")
	}
}

// TestConventionalKeyFieldBreaksTheTie pins the preference order: among
// several fields of the key's type, ID outranks everything, because that is
// the name an author gives the field that is the identity.
func TestConventionalKeyFieldBreaksTheTie(t *testing.T) {
	t.Parallel()

	s := kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
		field{"Name", "string"}, field{"ID", "string"})
	b := bindingsOf(t, s)
	testkit.Equal(t, b.Reference.KeyField, "ID", "the conventional spelling wins")
}

// field is one struct field of a kvStore fixture.
type field struct{ name, typ string }

// kvStore declares a reader over readV and a writer taking writeV, plus the
// Doc struct where either names it — the smallest interface the reference
// derivation walks all the way.
func kvStore(t *testing.T, readV, writeV string, fields ...field) *sdk.Store {
	t.Helper()
	return kvStoreWith(t, readV, writeV, nil, fields...)
}

// kvStoreWith is [kvStore] with directive options, for the supplied-reference
// arms.
func kvStoreWith(
	t *testing.T,
	readV, writeV string,
	opts []storefixture.DirectiveOption,
	fields ...field,
) *sdk.Store {
	t.Helper()
	f := storefixture.New().Package("kv", "example.com/kv")
	if len(fields) > 0 {
		f = f.Struct("Doc", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("kv/iface.go", 1, 1))
			for _, fl := range fields {
				b.Field(fl.name, typeOf(fl.typ), nil)
			}
		})
	}
	s := f.Interface("Store", func(i *storefixture.InterfaceBuilder) {
		i.Pos(sdk.At("kv/iface.go", 1, 1))
		i.Directive(storefixture.Directive("suite"))
		i.Directive(storefixture.Directive("model", opts...))
		i.Method("Get", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("key", storefixture.Named("string"))
			m.Return(typeOf(readV))
			m.Return(storefixture.Named("error"))
		})
		i.Method("Put", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", typeOf(writeV))
			m.Return(storefixture.Named("error"))
		})
	}).Build()
	stampShape(s, "Get", "reader", "string", readV)
	stampShape(s, "Put", "writer", "", writeV)
	return s
}

// typeOf spells a fixture type from its stamp form.
func typeOf(q string) *sdk.TypeRef {
	if idx := strings.LastIndexByte(q, '.'); idx >= 0 {
		return storefixture.PkgNamed(q[:idx], q[idx+1:])
	}
	return storefixture.Named(q)
}

// bindingsOf runs suite then model over the store and returns what model
// queued, failing where it queued nothing.
func bindingsOf(t *testing.T, s *sdk.Store) *model.Bindings {
	t.Helper()
	generateBoth(t, s)
	for _, p := range s.Emit().PendingOriginSlots() {
		if b, ok := p.Item.(*model.Bindings); ok {
			return b
		}
	}
	t.Fatal("the run queued no bindings")
	return nil
}

// generateBoth runs the two generators in bucket order, the way the pipeline
// does: model reads the projection suite queues.
func generateBoth(t *testing.T, s *sdk.Store) *diag.Sink {
	t.Helper()
	plugintest.Generate(t, suite.New(), s)
	return plugintest.Generate(t, model.New(), s)
}

// mixed is the corpus fixture in store form, stamped the way the annotator
// stamps it: a writer carrying the validates mixin, the validator it names,
// and a reader.
// genericFixture is a one-parameter generic store, its model directive
// carrying whatever the case supplies — the witness key or nothing.
func genericFixture(t *testing.T, opts ...storefixture.DirectiveOption) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("gen", "example.com/gen").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("gen/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model", opts...))
			i.TypeParam("V", nil)
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.TypeParamRef("V"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Param("v", storefixture.TypeParamRef("V"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()

	// Stamped the way the annotator stamps a generic pair: the parameter's
	// bare name is the value spelling, which is exactly what the witness
	// substitution rewrites.
	stampShape(s, "Get", "reader", "string", "V")
	stampShape(s, "Put", "compositewriter", "string", "V")
	return s
}

func mixed(t *testing.T, opts ...storefixture.DirectiveOption) *sdk.Store {
	t.Helper()
	return mixedWith(t, nil, opts...)
}

// mixedWith is [mixed] plus extra methods, for the fixtures that probe what an
// unmappable method does to an otherwise ordinary interface.
func mixedWith(
	t *testing.T,
	extra func(i *storefixture.InterfaceBuilder),
	opts ...storefixture.DirectiveOption,
) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("validates", "example.com/validates").
		Struct("Payload", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("validates/iface.go", 1, 1))
			b.Field("Key", storefixture.Named("string"), nil)
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("validates/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model", opts...))
			i.Method("Store", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("v", storefixture.PkgNamed("example.com/validates", "Payload"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Validate", func(m *storefixture.MethodBuilder) {
				m.Param("v", storefixture.PkgNamed("example.com/validates", "Payload"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Read", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.PkgNamed("example.com/validates", "Payload"))
				m.Return(storefixture.Named("error"))
			})
			if extra != nil {
				extra(i)
			}
		}).
		Build()

	stampShape(s, "Store", "writer", "", "example.com/validates.Payload")
	stampShape(s, "Read", "reader", "string", "example.com/validates.Payload")
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Store" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{"validates"}, "test")
			shape.MixinParamKey("validates", "fn").
				Set(bag, "example.com/validates.Validate", "test")
		}
	}
	return s
}

// readerOnly declares one stamped reader — nothing for a map to model.
func readerOnly(t *testing.T) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("ro", "example.com/ro").
		Interface("Fetcher", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("ro/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			i.Method("Fetch", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	stampShape(s, "Fetch", "reader", "string", "string")
	return s
}

// stampShape sets what the annotator would have written for one method.
func stampShape(s *sdk.Store, method, shapeName, keyType, valueType string) {
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != method {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaShape.Set(bag, shapeName, "test")
			if keyType != "" {
				shape.MetaKeyType.Set(bag, keyType, "test")
			}
			if valueType != "" {
				shape.MetaValueType.Set(bag, valueType, "test")
			}
		}
	}
}

// TestDrainFixtures walks the writer-plus-collector fork both ways: a keyed
// value upserts and gets the map, a bare value appends and gets the
// collection — deduplicating exactly where the claim says so.
func TestDrainFixtures(t *testing.T) {
	t.Parallel()

	t.Run("a keyed value selects the map", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, drainStore(t, true))
		testkit.Equal(t, b.Reference.StoreType(), "MapStore",
			"an ID or Key field means upsert semantics")
		testkit.Equal(t, b.Reference.KeyField, "Key", "keyed on the conventional field")
		testkit.False(t, b.Reference.Collects(), "the map is not a collection")

		ops := map[string]string{}
		for _, am := range b.Adapter {
			ops[am.Sig.Name] = am.Op
		}
		testkit.Equal(t, ops["Items"], "Values", "the collector drains the map's values")

		testkit.True(t, b.Values.Wide, "the walk recurses through the nested struct")
		testkit.Equal(t, b.Values.Pin, "Key", "pinned on the upsert field")
		testkit.Equal(t, b.Keys.Field, b.Values.Field+".Key",
			"with no reader, the fixture values' own keys are the colliding set")
		testkit.True(t, b.UsesKeys(), "which the pin draws from")

		var bound *model.LawBinding
		for _, l := range b.Laws {
			if l.ID == lawid.StreamNoDuplicates {
				bound = l
			}
		}
		testkit.True(t, bound != nil, "the no-duplicates law binds")
		testkit.Equal(t, bound.Fields[0].Kind(), bound.Fields[0].KindName,
			"its drain renders through the field's template")
	})

	t.Run("a bare value selects the collection, deduplicating by claim", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, drainStore(t, false))
		testkit.True(t, b.Reference.Collects(), "no identity field, so append-and-drain")
		testkit.Equal(t, b.Reference.StoreType(), "SetCollection",
			"the no-duplicates claim refines the log into a set")
	})
}

// TestDrainMismatchFallsToTheTwin pins the collection fork's one agreement:
// the writer adds what the collector returns, or no collection models the
// pair and the twin floor stands in.
func TestDrainMismatchFallsToTheTwin(t *testing.T) {
	t.Parallel()

	s := drainStore(t, false)
	stampShape(s, "Add", "writer", "", "example.com/bag.Other")
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Twin(), "one collection cannot model the pair")
	testkit.Assert(t, b.Reference.TwinWhy).Contains("the writer adds",
		"and the header spells the disagreement")
}

// TestCompanionSurface reaches the second emit value the plugin queues.
func TestCompanionSurface(t *testing.T) {
	t.Parallel()

	s := mixed(t)
	generateBoth(t, s)
	var comp *model.Companion
	for _, p := range s.Emit().PendingOriginSlots() {
		if c, ok := p.Item.(*model.Companion); ok {
			comp = c
		}
	}
	testkit.True(t, comp != nil, "the proof rides with the bindings")
	testkit.Equal(t, comp.Kind(), model.KindCompanion, "and renders as itself")
	testkit.Equal(t, comp.ModelPkg(), model.ModelPkg, "reaching the runner's package")
	comp.SetOutputPackages(map[string]string{"": "example.com/validates/validatestest"})
	testkit.Equal(t, comp.HarnessPkg, "example.com/validates/validatestest",
		"and the bindings through Layout's resolved route")
	comp.SetOutputPackages(map[string]string{})
	testkit.Equal(t, comp.HarnessPkg, "example.com/validates/validatestest",
		"which a partial later map does not clear")

	testkit.Equal(t, len(comp.Mutants), 2,
		"one kill-matrix row per driven method")
	testkit.Equal(t, comp.Mutants[0].Method, "Store", "the writer's mutant leads")
	testkit.Equal(t, comp.Mutants[1].Method, "Read", "the reader's follows")
	testkit.True(t, comp.Mutants[0].Sig != nil, "each with the override's signature")
	testkit.Equal(t, comp.ConcurrentName, "MixedModelConcurrent",
		"and the concurrent leg's proof rides along where the leg derives")
	testkit.Equal(t, comp.RootPkg(), "go.thesmos.sh/testkit",
		"the surrogate's import path reaches the template")
}

// drainStore is the writer-plus-collector fixture: Add and Items, the
// noduplicates claim on the collector, and a value type with or without the
// conventional identity field.
func drainStore(t *testing.T, keyedValue bool) *sdk.Store {
	t.Helper()
	// The decoy is what makes the struct search walk past a non-match: a
	// store with exactly one struct never exercises the mismatch arm.
	f := storefixture.New().Package("bag", "example.com/bag").
		Struct("Decoy", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("bag/iface.go", 1, 1))
			b.Field("N", storefixture.Named("int"), nil)
		})
	valueRef := storefixture.Named("string")
	valueQ := "string"
	if keyedValue {
		// The two Decoy fields make the wide-draw walk recurse into a nested
		// struct and revisit it — the diamond that exercises the seen set.
		f = f.Struct("Value", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("bag/iface.go", 1, 1))
			b.Field("Key", storefixture.Named("string"), nil)
			b.Field("Body", storefixture.Named("string"), nil)
			b.Field("Meta", storefixture.PkgNamed("example.com/bag", "Decoy"), nil)
			b.Field("More", storefixture.PkgNamed("example.com/bag", "Decoy"), nil)
		})
		valueRef = storefixture.PkgNamed("example.com/bag", "Value")
		valueQ = "example.com/bag.Value"
	}
	s := f.Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
		i.Pos(sdk.At("bag/iface.go", 1, 1))
		i.Directive(storefixture.Directive("suite"))
		i.Directive(storefixture.Directive("model"))
		i.Method("Add", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", valueRef)
			m.Return(storefixture.Named("error"))
		})
		i.Method("Items", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Return(storefixture.Slice(valueRef))
			m.Return(storefixture.Named("error"))
		})
	}).Build()
	stampShape(s, "Add", "writer", "", valueQ)
	stampShape(s, "Items", "aggregator", "", "")
	// The claim rides on both halves: the drain carries the law, and the
	// writer exercises the second of the two mixin scans the dedupe
	// refinement makes.
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == "Add" || m.Name == "Items" {
				shape.MetaMixins.Set(m.EnsureMeta(), []string{"noduplicates"}, "test")
			}
		}
	}
	return s
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
			i.Pos(sdk.At("zoo/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			method := func(name string, params, returns []*sdk.TypeRef) {
				i.Method(name, func(m *storefixture.MethodBuilder) {
					m.Param("ctx", storefixture.PkgNamed("context", "Context"))
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
				m.Return(storefixture.Named("error"))
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

// TestOracleDefeatingClaims pins the eventually arm: reads may lag writes, so
// every immediate oracle mis-models the subject and the twins stand in.
func TestOracleDefeatingClaims(t *testing.T) {
	t.Parallel()

	s := mixed(t)
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == "Read" {
				shape.MetaMixins.Set(m.EnsureMeta(), []string{"eventually"}, "test")
			}
		}
	}
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Twin(), "no immediate store models the lag")
	testkit.Assert(t, b.Reference.TwinWhy).Contains("lag",
		"and the header names the claim's slack")
}

// TestHistoryDrainForcesTheLog pins the drain fork's history arm: a claim
// whose vocabulary is events outranks the upsert inference an incidental Key
// field would trigger.
func TestHistoryDrainForcesTheLog(t *testing.T) {
	t.Parallel()

	s := drainStore(t, true)
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			// Replacing the fixture's noduplicates claim outright: a history
			// under a dedupe claim is a different fixture's question.
			shape.MetaMixins.Set(m.EnsureMeta(), []string{"snapshotisolation"}, "test")
		}
	}
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Collects(), "events append; they do not upsert")
	testkit.False(t, b.Reference.Dedupe, "and identical events repeat")
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
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Return(storefixture.Slice(storefixture.Named("string")))
			m.Return(storefixture.Named("error"))
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

// TestConcurrentLeg pins where the linearizability leg derives — the
// unrefined map pair, whose reader and writer the Porcupine keyed-store
// model speaks — and where it must not: a pin changes what a read means, a
// keyed put carries its key outside the value, and a twin has no model at
// all.
func TestConcurrentLeg(t *testing.T) {
	t.Parallel()

	t.Run("the map pair derives it over the sequential actions", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
			field{"ID", "string"}))
		testkit.True(t, b.Concurrent(), "reader and writer, one map, no refinement")
		testkit.Equal(t, b.ConcReader.Method, "Get", "the reader leg is the sequential reader")
		testkit.Equal(t, b.ConcWriter.Method, "Put", "and the writer leg the sequential writer")
	})

	t.Run("a pinning map does not", func(t *testing.T) {
		t.Parallel()
		s := kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc", field{"ID", "string"})
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Get" {
					shape.MetaMixins.Set(m.EnsureMeta(), []string{"sticky"}, "test")
				}
			}
		}
		b := bindingsOf(t, s)
		testkit.False(t, b.Concurrent(), "a pinned read is not the keyed-store model's read")
	})

	t.Run("the keyed store does not", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, keyedStore(t, "example.com/kv.ErrGone"))
		testkit.False(t, b.Concurrent(), "the keyed put carries no key projection to partition by")
	})

	t.Run("the twin does not", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, kvStore(t, "string", "string"))
		testkit.False(t, b.Concurrent(), "a twin has no linearizability model to check against")
	})
}

// TestGenSupply pins the gen= key: the consumer's generator constructor
// replaces reflection as the wide arm — outranking every narrowing verdict,
// because the consumer authored the domain — and a qualified spelling is
// refused the way ref='s is.
func TestGenSupply(t *testing.T) {
	t.Parallel()

	s := mixed(t, storefixture.KV(model.GenKey, "PayloadGen"))
	b := bindingsOf(t, s)
	testkit.Equal(t, b.Values.GenFunc, "PayloadGen", "the supply is recorded")
	testkit.True(t, b.Values.Wide, "and the pool goes wide through it")

	t.Run("a qualified constructor is refused", func(t *testing.T) {
		t.Parallel()
		s := mixed(t, storefixture.KV(model.GenKey, "other.PayloadGen"))
		got := generateBoth(t, s).Diagnostics()
		testkit.Equal(t, len(got), 1, "one diagnostic")
		testkit.Assert(t, got[0].Message).Contains("routed output package",
			"naming where the constructor must live")
	})
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
			i.Pos(sdk.At("fn/iface.go", 1, 1))
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

// TestContractOracle walks the role-stamped derivation over the lease
// family: the carrier's role= names its own part, the partner key names the
// sibling, the store's type argument is the acquire key's, and the
// constructor's sentinels follow the claims — minted where the strict
// dialect holds, lenified to the twin where every sentinel would be nil,
// because the kill matrix proved a never-disagreeing oracle checks nothing.
func TestContractOracle(t *testing.T) {
	t.Parallel()

	leased := func(t *testing.T, idempotent bool) *sdk.Store {
		t.Helper()
		return leasedWith(t, idempotent, nil)
	}

	t.Run("the strict dialect derives the tracker", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, leased(t, false))
		testkit.True(t, b.Reference.IsContract(), "the claim outranks the shapes")
		testkit.Equal(t, b.Reference.StoreType(), "LeaseTracker", "to its own store")
		ops := map[string]string{}
		for _, am := range b.Adapter {
			ops[am.Sig.Name] = am.Op
		}
		testkit.Equal(t, ops["Acquire"], "Acquire", "the carrier delegates by role")
		testkit.Equal(t, ops["Release"], "Release", "and the partner beside it")
		testkit.Equal(t, len(b.Reference.CtorErrs), 2, "two constructor slots")
		testkit.True(t, b.Reference.CtorErrs[0].Name != "", "the held sentinel is minted")
		testkit.Equal(t, b.Reference.CtorErrs[1].Name, "", "and the release slot stays lenient")
	})

	t.Run("the idempotent claim lenifies to the twin", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, leased(t, true))
		testkit.True(t, b.Reference.Twin(), "a never-disagreeing oracle checks nothing")
		testkit.Assert(t, b.Reference.TwinWhy).Contains("lenify",
			"and the header says which claims did it")
	})

	t.Run("non-role methods stay inert on the contract oracle", func(t *testing.T) {
		t.Parallel()
		s := leasedWith(t, false, func(i *storefixture.InterfaceBuilder) {
			i.Method("Count", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Return(storefixture.Named("int"))
				m.Return(storefixture.Named("error"))
			})
		})
		stampShape(s, "Count", "aggregator", "", "int")
		b := bindingsOf(t, s)
		testkit.True(t, b.Reference.IsContract(), "the roles still resolve")
		inert := map[string]string{}
		for _, am := range b.Adapter {
			if am.Op == "" {
				inert[am.Sig.Name] = am.Reason
			}
		}
		testkit.Assert(t, inert["Count"]).Contains("only its roles",
			"a method outside the vocabulary is inert, with why")
	})

	t.Run("selection deduplicates across the role carriers", func(t *testing.T) {
		t.Parallel()
		// The classification rides every role method; re-selecting its laws
		// from each carrier must not register one law twice or print one
		// refusal per method.
		s := leasedWith(t, false, nil)
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Release" {
					shape.MetaContracts.Set(m.EnsureMeta(), []string{"lease"}, "test")
				}
			}
		}
		b := bindingsOf(t, s)
		seen := map[string]int{}
		for _, u := range b.Unbound {
			seen[u.Method+"\x00"+u.Reason]++
		}
		for key, n := range seen {
			testkit.Equal(t, n, 1, "one refusal per (law, reason): "+key)
		}
	})

	t.Run("a partner naming nothing leaves the role unresolved", func(t *testing.T) {
		t.Parallel()
		s := leasedWith(t, false, nil)
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Acquire" {
					shape.ContractPartnerKey("lease", "release").
						Set(m.EnsureMeta(), "example.com/ls.Nowhere", "test")
				}
			}
		}
		b := bindingsOf(t, s)
		testkit.False(t, b.Reference.IsContract(),
			"a role resolved to nothing derives nothing")
	})

	t.Run("a carrier without a role stamp falls through", func(t *testing.T) {
		t.Parallel()
		s := leasedWith(t, false, nil)
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Acquire" {
					shape.ContractRoleKey("lease").Set(m.EnsureMeta(), "", "test")
				}
			}
		}
		got := bindingsOf(t, s)
		testkit.False(t, got.Reference.IsContract(),
			"no role, no delegation to hang the oracle on")
	})

	t.Run("a type-arg role speaking no type falls through", func(t *testing.T) {
		t.Parallel()
		s := storefixture.New().
			Package("ls", "example.com/ls").
			Interface("Locker", func(i *storefixture.InterfaceBuilder) {
				i.Pos(sdk.At("ls/iface.go", 1, 1))
				i.Directive(storefixture.Directive("suite"))
				i.Directive(storefixture.Directive("model"))
				// The acquire role takes nothing, so the tracker's type
				// argument has nowhere to come from.
				i.Method("Acquire", func(m *storefixture.MethodBuilder) {
					m.Param("ctx", storefixture.PkgNamed("context", "Context"))
					m.Return(storefixture.Named("error"))
				})
				i.Method("Release", func(m *storefixture.MethodBuilder) {
					m.Param("ctx", storefixture.PkgNamed("context", "Context"))
					m.Return(storefixture.Named("error"))
				})
			}).
			Build()
		stampShape(s, "Acquire", "lifecycle", "", "")
		stampShape(s, "Release", "lifecycle", "", "")
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name != "Acquire" {
					continue
				}
				bag := m.EnsureMeta()
				shape.MetaContracts.Set(bag, []string{"lease"}, "test")
				shape.ContractRoleKey("lease").Set(bag, "acquire", "test")
				shape.ContractPartnerKey("lease", "release").
					Set(bag, "example.com/ls.Release", "test")
			}
		}
		got := bindingsOf(t, s)
		testkit.False(t, got.Reference.IsContract(),
			"an oracle with no type argument cannot be instantiated")
	})

	t.Run("an unresolved role falls through to the shapes", func(t *testing.T) {
		t.Parallel()
		s := leased(t, false)
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Acquire" {
					shape.ContractPartnerKey("lease", "release").Set(m.EnsureMeta(), "", "test")
				}
			}
		}
		b := bindingsOf(t, s)
		testkit.False(t, b.Reference.IsContract(),
			"half a lease checks nothing a twin does not")
	})
}

// leasedWith builds the lease-contract fixture: a stamped carrier, its
// release partner, and whatever extra methods a probe needs.
func leasedWith(
	t *testing.T,
	idempotent bool,
	extra func(i *storefixture.InterfaceBuilder),
) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("ls", "example.com/ls").
		Interface("Locker", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("ls/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			for _, name := range []string{"Acquire", "Release"} {
				i.Method(name, func(m *storefixture.MethodBuilder) {
					m.Param("ctx", storefixture.PkgNamed("context", "Context"))
					m.Param("key", storefixture.Named("string"))
					m.Return(storefixture.Named("error"))
				})
			}
			if extra != nil {
				extra(i)
			}
		}).
		Build()
	stampShape(s, "Acquire", "writer", "", "string")
	stampShape(s, "Release", "writer", "", "string")
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Acquire" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaContracts.Set(bag, []string{"lease"}, "test")
			shape.ContractRoleKey("lease").Set(bag, "acquire", "test")
			shape.ContractPartnerKey("lease", "release").
				Set(bag, "example.com/ls.Release", "test")
			if idempotent {
				shape.MetaMixins.Set(bag, []string{"idempotent"}, "test")
			}
		}
	}
	return s
}
