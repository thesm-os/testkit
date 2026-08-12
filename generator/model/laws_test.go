// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
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

// TestUnboundLawIsReported holds the miss to visibility: a selected law the
// catalogue cannot instantiate is named in the header with what it waits on,
// never silently absent from the registry.
func TestUnboundLawIsReported(t *testing.T) {
	t.Parallel()

	s := mixedWith(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("List", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Return(storefixture.PkgNamed("example.com/validates", "Payload"))
			m.Return(storefixture.Named("error"))
		})
	})
	stampShape(s, "List", "streamreader", "", "example.com/validates.Payload")
	b := bindingsOf(t, s)

	testkit.Equal(t, len(b.Laws), 1, "the writer's law still binds")

	// The streamreader's laws select and refuse: List streams through an
	// iterator, and the drain spelling returns a slice. Selected-and-refused
	// is a header line; selected-and-miscompiled would be a consumer's build.
	unbound := map[string]string{}
	for _, u := range b.Unbound {
		unbound[u.Method] = u.Reason
	}
	testkit.Assert(t, unbound[lawid.StreamCompletion]).Contains("iterator",
		"the drain names what it cannot yet consume")
	testkit.Assert(t, unbound[lawid.StreamReentrant]).Contains("iterator",
		"and so does the reentrancy claim")
}

// TestNoReaderWriterPairAtAll covers the other half of the twin floor: a
// writer with no reader derives no store either, and arms as twins.
func TestNoReaderWriterPairAtAll(t *testing.T) {
	t.Parallel()

	s := storefixture.New().
		Package("wo", "example.com/wo").
		Interface("Sink", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("wo/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("model"))
			i.Method("Push", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("v", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	stampShape(s, "Push", "writer", "", "string")

	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Twin(), "no store derives, so the twins stand in")
	testkit.True(t, b.UsesValues() && !b.UsesKeys(),
		"and the writer still draws from its pool")
}
