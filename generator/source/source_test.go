// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package source_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/source"
)

// why is a caller's consequence, spelled so a failure names which
// clause reached the diagnostic.
var why = source.Consequence{
	Partial:    "the artifact covers what the source contributed",
	Incomplete: "a partial artifact asserts less than it appears to",
}

// A resolvable interface is complete and says nothing.
//
// The path every corpus fixture takes, and the one a regression here
// would break silently: a generator that stopped trusting `complete`
// would emit nothing for a source with no problem at all.
func TestMethodSetResolves(t *testing.T) {
	t.Parallel()

	ctx, iface := fixture(t, func(b *storefixture.InterfaceBuilder) {
		b.Method("Put", nil)
	})

	set, complete := source.MethodSet(ctx, iface, "test", why)
	testkit.True(t, complete, "an interface embedding nothing is complete")
	testkit.Len(t, set.Methods, 1, "and carries its own method")
	testkit.Len(t, ctx.Diag.Diagnostics(), 0, "a source with no issue earns no diagnostic")
}

// An embed this run did not load refuses the interface, as a warning.
//
// Warning rather than error because the same source under a wider
// pattern resolves: the author wrote nothing wrong, the run was too
// narrow. Generation still stops — the whole point of the flag — so
// this asserts both halves, since a warning that let generation
// proceed would emit an artifact over half a contract.
func TestMethodSetRefusesAnUnloadedEmbed(t *testing.T) {
	t.Parallel()

	ctx, iface := fixture(t, func(b *storefixture.InterfaceBuilder) {
		b.Method("Put", nil)
		b.Embed(storefixture.PkgNamed("elsewhere", "Reader"))
	})

	_, complete := source.MethodSet(ctx, iface, "test", why)
	testkit.False(t, complete, "an unresolvable embed stops generation")

	diags := ctx.Diag.Diagnostics()
	testkit.Len(t, diags, 1, "and says so once")
	testkit.Contains(t, diags[0].Message, why.Incomplete,
		"ending on the caller's own consequence, not a generic one")
	testkit.Contains(t, diags[0].Message, "Reader",
		"and naming the embed the author can search for")
}

// fixture builds a one-interface store and the context a generator
// reads it through.
func fixture(t *testing.T, build func(*storefixture.InterfaceBuilder)) (*sdk.GeneratorContext, *sdk.Interface) {
	t.Helper()

	store := storefixture.New().
		Package("subject", "example.com/subject").
		Interface("Subject", build).
		Build()

	ctx := &sdk.GeneratorContext{Store: store, Reader: sdk.NewStoreReader(store), Diag: sdk.NewSink()}
	iface, found := ctx.Reader.Interfaces().Where(func(i *sdk.Interface) bool {
		return i.Name == "Subject"
	}).First()
	testkit.True(t, found, "the fixture declares the interface under test")
	return ctx, iface
}

// The companion is found by name and confirmed by signature.
//
// Both halves matter: the name alone would accept a `PayloadDefaults`
// taking arguments, and calling that emits a constructor the consumer
// cannot build. The refusal is silent by design — a type with no
// companion is the ordinary case, not a fault — so the nil return is
// what a caller reads, and it is asserted rather than assumed.
func TestCompanion(t *testing.T) {
	t.Parallel()

	t.Run("a well-formed companion resolves", func(t *testing.T) {
		t.Parallel()
		ctx := withFunc(t, "PayloadDefaults", storefixture.Named("Payload"))
		got := source.Companion(ctx, "example.com/subject", "Payload", "Defaults")
		testkit.True(t, got != nil, "the convention names a function this run loaded")
	})

	t.Run("a colliding name with the wrong return is declined", func(t *testing.T) {
		t.Parallel()
		ctx := withFunc(t, "PayloadDefaults", storefixture.Named("Other"))
		testkit.True(t, source.Companion(ctx, "example.com/subject", "Payload", "Defaults") == nil,
			"a function answering another type is a collision, not a companion")
	})

	t.Run("no companion is not a fault", func(t *testing.T) {
		t.Parallel()
		ctx := withFunc(t, "Unrelated", storefixture.Named("Payload"))
		testkit.True(t, source.Companion(ctx, "example.com/subject", "Payload", "Defaults") == nil,
			"the ordinary case answers nil and says nothing")
		testkit.Len(t, ctx.Diag.Diagnostics(), 0, "a type without one is not diagnosed")
	})
}

// withFunc builds a store declaring one nullary function returning ret.
func withFunc(t *testing.T, name string, ret *node.TypeRef) *sdk.GeneratorContext {
	t.Helper()

	store := storefixture.New().
		Package("subject", "example.com/subject").
		Function(name, func(b *storefixture.FunctionBuilder) { b.Return(ret) }).
		Build()
	return &sdk.GeneratorContext{Store: store, Reader: sdk.NewStoreReader(store), Diag: sdk.NewSink()}
}
