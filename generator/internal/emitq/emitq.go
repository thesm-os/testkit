// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package emitq queues emit values against the source declaration they were
// projected from.
//
// Every generator here ends the same way: build one value per output, stamp
// each with provenance naming the plugin and the value's kind, and append it to
// a slot on the origin node so layout can route it. The steps are identical
// whatever is being generated — only the values differ — and each generator
// that wrote them out again wrote the same four lines and the same comment
// explaining why they are a loop rather than two blocks.
//
// The cost of that was not the duplication. It was that the one generator which
// factored it out privately drifted: it grew a second construction of the emit
// base that set the fields in a different order, and nothing could tell the two
// apart. Queueing is framework mechanics, not a generator's opinion, so it lives
// once.
package emitq

import (
	"fmt"

	"go.thesmos.sh/eidos/core/contract"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"
)

// Base builds the emit base for a value projected from origin.
//
// The three fields travel together — the node the value came from, the plugin
// that made it, and the position a diagnostic about it should point at — and a
// value missing any of them is one whose failures name the wrong source line.
func Base(c *sdk.Provenance, origin node.Node) sdk.BaseEmit {
	return sdk.BaseEmit{
		OriginNode: origin,
		SetByName:  c.SetBy(),
		SourcePos:  origin.Pos(),
	}
}

// Tagged returns base routed to the named output tag.
//
// Returned by value rather than mutated in place: a caller building a primary
// and a companion holds one base and derives the second from it, and a helper
// that mutated would leave the first pointing at the companion's output.
func Tagged(base sdk.BaseEmit, tag string) sdk.BaseEmit {
	base.OutputTagName = tag
	return base
}

// Append queues every value against origin's slot, stamping each with
// provenance naming its kind.
//
// The provenance id is `<kind>.<origin>`, which is what a later plugin targets
// when it positions its own contribution relative to this one, and what a
// reader chasing "which plugin wrote this line" gets back from `testkit
// explain`.
//
// Takes the values variadically because a generator's outputs are a set that
// grows: appending them in one call is what keeps the primary and its companion
// from acquiring separate, divergent copies of this logic — which is how the
// two ended up stamped with different provenance in the first place.
func Append(
	ctx *sdk.GeneratorContext,
	c *sdk.Provenance,
	slot string,
	origin contract.Owner,
	values ...sdk.EmitNode,
) error {
	return AppendAs(ctx, c, slot, origin, origin.OwnerName(), values...)
}

// AppendAs is [Append] for a value whose provenance names something other than
// the declaration it hangs off.
//
// Package-scoped output has no declaration of its own, so it is anchored on one
// the package happens to contain and identified by the package it is really
// about. Deriving the id from the anchor there would make the identifier a
// plugin targets depend on which declaration sorted first, and renaming an
// unrelated type in the package would move it.
func AppendAs(
	ctx *sdk.GeneratorContext,
	c *sdk.Provenance,
	slot string,
	origin node.Node,
	id string,
	values ...sdk.EmitNode,
) error {
	for _, value := range values {
		prov := c.Provenance(string(value.Kind()) + "." + id)
		if err := ctx.Store.Emit().AppendOriginSlot(origin, slot, value, prov); err != nil {
			return fmt.Errorf("%s: append %s slot for %q: %w", c.SetBy(), value.Kind(), id, err)
		}
	}
	return nil
}

// PrimaryPackage returns the package a plugin's primary output routed to, and
// whether layout resolved one.
//
// Layout calls [emit.OutputPackageSetter] at most once, after every target
// resolves, and may pass a partial map: a run that recorded routing errors
// reaches dispatch with tags missing, and the primary tag can be present but
// empty. Both are the same answer to a caller — there is no package to qualify
// against — and folding them here is what stops each implementation reasoning
// about it again and one of them getting it wrong.
//
// A caller that skips on false leaves its provisional references in place. That
// is deliberate: a reference qualified against the wrong package is a compile
// error naming the symbol, which is a better failure than a bare name silently
// binding to whatever else is in scope.
func PrimaryPackage(byTag map[string]string) (string, bool) {
	path, ok := byTag[""]
	if !ok || path == "" {
		return "", false
	}
	return path, true
}
