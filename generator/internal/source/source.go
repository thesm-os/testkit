// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package source answers questions about the source tree that a
// generator needs and no generator owns.
//
// Plugins import each other freely here — fault reads stub, model reads
// suite — so a helper with an owner belongs with its owner and is
// exported from there. These have none: resolving an interface's method
// set is not the harness generator's concept any more than the double
// generator's, and finding the companion beside a type is not the
// builder's just because the builder's directive can override it.
// Each was carried in two plugins before it was carried here.
package source

import (
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
)

// Consequence is the clause a generator's diagnostics end on — what its
// output loses when an embed cannot be resolved.
//
// Carried as data because it is the only thing that differs between
// callers, and it differs for a real reason: a harness over part of a
// contract passes an implementation that fails the rest, while a double
// missing a method cannot stand in at all. Both are refusals; a reader
// deserves the one that applies to what they asked for.
type Consequence struct {
	// Partial ends the cycle diagnostic, which is a warning: the walk
	// broke the cycle only after the interface it points back at had
	// contributed, so the set is short of nothing.
	Partial string

	// Incomplete ends the two diagnostics that stop generation.
	Incomplete string
}

// MethodSet returns the interface's full method set and whether it is
// complete.
//
// Resolution itself is [sdk.StoreReader.MethodSet]: the embed walk, the
// duplicate rule, the cycle guard and the attribution of a method to
// the embed it arrived through are facts about a Go method set. What is
// decided here is what a generator does with an incomplete one —
// refuse, because emitting over part of a contract reports success for
// an implementation that satisfies none of the rest.
//
// Severity splits on whether a wider run would fix it: an unloaded
// embed is a warning, because the same source under a wider pattern
// resolves and the author has done nothing wrong. Anything else is an
// error against the declaration.
func MethodSet(
	ctx *sdk.GeneratorContext, iface *sdk.Interface, plugin string, why Consequence,
) (sdk.MethodSetResult, bool) {
	set := ctx.Reader.MethodSet(iface)
	complete := true
	for _, issue := range set.Issues {
		// Spelled the way the source wrote it — `io.Closer`, not the bare
		// `Closer` the reference carries — so a diagnostic names something
		// the author can search for.
		written := golang.Display(issue.Embed.Type)
		switch issue.Reason {
		case sdk.ReasonCyclic:
			ctx.Diag.Warnf(issue.Embed.Pos(),
				"%s: interface %q embeds %q through a cycle; the walk broke out of it, "+
					"so %s",
				plugin, iface.QName(), written, why.Partial)
		case sdk.ReasonUnresolved:
			complete = false
			ctx.Diag.Warnf(issue.Embed.Pos(),
				"%s: interface %q embeds %q, which this run did not load, so its method "+
					"set cannot be completed; nothing is generated, because %s",
				plugin, iface.QName(), written, why.Incomplete)
		default:
			complete = false
			ctx.Diag.Errorf(issue.Embed.Pos(),
				"%s: interface %q embeds %q, which %s; nothing is generated, because %s",
				plugin, iface.QName(), written, issue.Reason, why.Incomplete)
		}
	}
	return set, complete
}

// Companion finds the `<Type><suffix>` function declared beside a type,
// nil where none is or where the one found is a different function that
// happens to collide.
//
// The suffix is a parameter because the convention belongs to whichever
// directive spells it, while the walk does not: the builder generator
// resolves a struct's defaults through it and the suite generator draws
// a fixture value through it, and both carried this same walk before it
// was carried here.
//
// The signature is checked rather than only the name: a `UserDefaults`
// taking arguments, or returning something else, is a different
// function, and calling it would emit code that does not compile.
func Companion(ctx *sdk.GeneratorContext, pkg, typeName, suffix string) *sdk.Expr {
	if typeName == "" {
		return nil
	}
	name := typeName + suffix
	fn, found := ctx.Reader.Functions().Where(func(fn *sdk.Function) bool {
		return fn.Name == name && fn.Package == pkg
	}).First()
	if !found || len(fn.Params) != 0 || len(fn.Returns) != 1 {
		return nil
	}
	if r := fn.Returns[0].Type; r == nil || r.Name != typeName {
		return nil
	}
	return sdk.NewExternal(pkg, name)
}
