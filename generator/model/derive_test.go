// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/plugins/annotator/shape"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/gentest"
	"go.thesmos.sh/testkit/generator/model"
	"go.thesmos.sh/testkit/generator/suite"
)

// TestUnmappableMethods holds the skip arms to their reasons: an unstamped
// method, one stamped outside the table, and one that cannot forward to the
// oracle are all listed — never silently absent from the sequences.
func TestUnmappableMethods(t *testing.T) {
	t.Parallel()

	s := mixedWith(t, func(i *storefixture.InterfaceBuilder) {
		// Unstamped: the annotator classified nothing.
		i.Method("Ping", func(m *storefixture.MethodBuilder) {
			gentest.Ctx(m)
			gentest.Err(m)
		})
		// Stamped with vocabulary this build's table has never heard of — the
		// state a detector landing upstream is in until the row lands.
		i.Method("Weird", func(m *storefixture.MethodBuilder) {
			gentest.Ctx(m)
			gentest.Err(m)
		})
		// Stamped and driven, but with no context to forward to the oracle.
		i.Method("Len", func(m *storefixture.MethodBuilder) {
			m.Return(storefixture.Named("int"))
			gentest.Err(m)
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
				i.Pos(gentest.AtFile("bare/iface.go"))
				i.Directive(storefixture.Directive("model"))
				i.Method("Get", func(m *storefixture.MethodBuilder) {
					gentest.Ctx(m)
					m.Param("key", storefixture.Named("string"))
					m.Return(storefixture.Named("string"))
					gentest.Err(m)
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
				i.Pos(gentest.AtFile("gen/iface.go"))
				i.Directive(storefixture.Directive("suite"))
				i.Directive(storefixture.Directive("model"))
				i.TypeParam("V", nil)
				i.Method("Put", func(m *storefixture.MethodBuilder) {
					gentest.Ctx(m)
					m.Param("v", storefixture.TypeParamRef("V"))
					gentest.Err(m)
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
				i.Pos(gentest.AtFile("un/iface.go"))
				i.Directive(storefixture.Directive("suite"))
				i.Directive(storefixture.Directive("model"))
				i.Method("Do", func(m *storefixture.MethodBuilder) {
					gentest.Ctx(m)
					gentest.Err(m)
				})
			}).
			Build()
		got := generateBoth(t, s).Diagnostics()
		testkit.Equal(t, len(got), 1, "one diagnostic")
		testkit.Assert(t, got[0].Message).Contains("drive",
			"sequences over nothing assert nothing")
	})
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
