// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/poisonable"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/internal/gentest"
	"go.thesmos.sh/testkit/generator/model"
)

// TestLawSelection walks the one law the corpus fixture's classifications
// earn, field by field, because the binding is the whole point of the tier:
// a wrong field here is a law that runs, passes, and asserts nothing.
func TestLawSelection(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, mixed(t))
	testkit.Equal(t, len(b.Laws), 1, "the writer's classification earns one law")

	l := b.Laws[0]
	testkit.Equal(t, l.ID, lawid.WriteObservable, "the identifier the header prints")
	testkit.Equal(t, len(l.Args), 3, "subject, value, key — in the struct's own order")

	names := make([]string, 0, len(l.Fields))
	for _, f := range l.Fields {
		names = append(names, f.Name)
	}
	testkit.Equal(t, names, []string{"Write", "Read", "Values", "KeyOf"},
		"every exported field is filled, in manifest order")
	testkit.Equal(t, l.Fields[0].Method, "Store", "Write closes over the writer")
	testkit.Equal(t, l.Fields[1].Method, "Read", "Read closes over the keyed reader")
	testkit.Equal(t, l.Fields[2].Pool, "values", "Values reuses the shared pool")
	testkit.Equal(t, l.Fields[3].KeyOfName, b.KeyOfName(),
		"KeyOf reuses the projection the reference is keyed on")
}

// TestIteratorStreamLawsBind pins the iterator adaptation: a streamreader
// drains through a collect loop rather than refusing, so its two laws bind at
// the stamped element type — the wall the audit's register carried, closed.
func TestIteratorStreamLawsBind(t *testing.T) {
	t.Parallel()

	s := mixedWith(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("List", func(m *storefixture.MethodBuilder) {
			gentest.Ctx(m)
			m.Return(storefixture.PkgNamed("example.com/validates", "Payload"))
			gentest.Err(m)
		})
	})
	stampShape(s, "List", "streamreader", "", "example.com/validates.Payload")
	b := bindingsOf(t, s)

	bound := map[string]*model.LawBinding{}
	for _, l := range b.Laws {
		bound[l.ID] = l
	}
	testkit.True(t, bound[lawid.WriteObservable] != nil, "the writer's law still binds")
	completion := bound[lawid.StreamCompletion]
	testkit.True(t, completion != nil, "the drain law binds over the iterator")
	if completion != nil {
		testkit.Equal(t, string(completion.Fields[0].Kind()), "model.lawfield.DrainSeq",
			"through the collect-loop template, not the slice spelling")
	}
	testkit.True(t, bound[lawid.StreamReentrant] != nil, "and so does the reentrancy claim")
}

// poisonProbe is the corpus poisonaccessor fixture in store form: a nullary
// bare-error probe, with the latch declared or not. One fixture for both
// tests, so the only variable between them is the claim.
func poisonProbe(t *testing.T, declared bool) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("poison", "example.com/poison").
		Interface("Probe", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("poison/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			i.Method("Err", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
			i.Method("Trip", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
			})
		}).
		Build()
	stampShape(s, "Err", "poisonaccessor", "", "")
	if declared {
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name != "Err" {
					continue
				}
				bag := m.EnsureMeta()
				shape.MetaMixins.Set(bag, []string{poisonable.Name}, "test")
				shape.MixinParamKey(poisonable.Name, poisonable.ParamInduce).
					Set(bag, "example.com/poison.Probe.Trip", "test")
			}
		}
	}
	return s
}

// TestPoisonNeedsTheDeclaredLatch pins what selects the poison pair. The
// signature is a nullary bare-error callable, which `Err`, `Close` and `Ping`
// all are — so selecting on the shape claimed every one of them, and the
// read-purity law failed every correct close-once teardown. The latch is a
// claim, not a shape, and `poisonable induce=` is where it is made.
func TestPoisonNeedsTheDeclaredLatch(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, poisonProbe(t, false))
	for _, l := range b.Laws {
		testkit.True(t, l.ID != lawid.PoisonNilOnFresh && l.ID != lawid.PoisonIdempotentRead,
			"an undeclared probe is a signature, not a latch")
	}
	for _, u := range b.Unbound {
		testkit.True(t, u.Method != lawid.PoisonNilOnFresh && u.Method != lawid.PoisonIdempotentRead,
			"and it is not a refusal either — the rule never selected")
	}
}

// TestPoisonBindsTheDeclaredLatch is the control: the same signature under
// the claim earns both laws.
func TestPoisonBindsTheDeclaredLatch(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, poisonProbe(t, true))

	bound := map[string]bool{}
	for _, l := range b.Laws {
		bound[l.ID] = true
	}
	testkit.True(t, bound[lawid.PoisonNilOnFresh], "the fresh-probe law binds")
	testkit.True(t, bound[lawid.PoisonIdempotentRead], "and the read-purity law binds")
}

// TestNoReaderWriterPairAtAll covers the other half of the twin floor: a
// writer with no reader derives no store either, and arms as twins.
func TestNoReaderWriterPairAtAll(t *testing.T) {
	t.Parallel()

	s := storefixture.New().
		Package("wo", "example.com/wo").
		Interface("Sink", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("wo/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			i.Method("Push", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("v", storefixture.Named("string"))
				gentest.Err(m)
			})
		}).
		Build()
	stampShape(s, "Push", "writer", "", "string")

	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Twin(), "no store derives, so the twins stand in")
	testkit.True(t, b.UsesValues() && !b.UsesKeys(),
		"and the writer still draws from its pool")
}
