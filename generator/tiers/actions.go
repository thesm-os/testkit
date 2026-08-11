// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import "slices"

// ActionFor returns the `engine/model/action` constructor that drives a method
// of the given detector shape, and whether the shape names one.
//
// The correspondence is data rather than derivation because the constructor
// names carry word boundaries the shape names flattened — `streamreader` is
// `Stream`, `poisonaccessor` is `PoisonCheck` — and it lives here rather than
// in the model generator because this is the module the conformance gate holds
// to both registries: every detector eidos registers must have a row, and
// every row must name a constructor the engine exports. A mapping the gate
// cannot see is how the last ownership table drifted to naming a third of what
// it claimed.
func ActionFor(shape string) (string, bool) {
	ctor, ok := actionCtors[shape]
	return ctor, ok
}

// actionCtors maps every detector shape to its action constructor.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var actionCtors = map[string]string{
	shapeAggregator:      "Aggregator",
	shapeBatchReader:     "BatchReader",
	shapeCompositeWriter: "CompositeWriter",
	shapeLifecycle:       "Lifecycle",
	shapeLookup:          "Lookup",
	shapeMultiAggregator: "MultiAggregator",
	shapeMultiArgWriter:  "MultiArgWriter",
	shapeMultiReader:     "MultiReader",
	shapeMutator:         "Mutator",
	shapePointerReader:   "PointerReader",
	shapePoisonAccessor:  "PoisonCheck",
	shapePredicate:       "Predicate",
	shapePure:            "Pure",
	shapeReader:          "Reader",
	shapeReaderNoError:   "ReaderNoError",
	shapeReaderWithBool:  "ReaderWithBool",
	shapeStreamConsumer:  "StreamConsumer",
	shapeStreamReader:    "Stream",
	shapeVoidLifecycle:   "VoidLifecycle",
	shapeWriter:          "Writer",
}

// MapStoreOp returns the [engine/model/ref.MapStore] method a shape delegates
// to, and whether the oracle models that shape at all.
//
// The oracle's methods were written to the shape signatures — `Get(ctx, K) (V,
// error)` is the reader shape verbatim — so a generated adapter forwards its
// parameters in order and changes nothing but the name. A shape with no row
// runs inert in the adapter, and the laws that would compare the subject
// against an inert body are not bound.
func MapStoreOp(shape string) (string, bool) {
	op, ok := mapStoreOps[shape]
	return op, ok
}

// MapStoreShapes returns every shape the oracle models, sorted.
//
// Exported for the two tests that close the table: the tiers side holds every
// key to the detector registry, and the conformance gate holds every value to
// a method the shipped oracle declares. Neither can iterate an unexported map.
func MapStoreShapes() []string {
	out := make([]string, 0, len(mapStoreOps))
	for s := range mapStoreOps {
		out = append(out, s)
	}
	slices.Sort(out)
	return out
}

// mapStoreOps maps the shapes MapStore models to its methods.
//
// The values spell the oracle's method names. Two also appear in the law
// catalogue as field names, and a shared constant would assert a unity that is
// not there — Put the oracle method and Put the law field are different
// things that happen to share a spelling, the catalogue's own stance one file
// over.
//
//nolint:gochecknoglobals,goconst // a lookup table, read-only after init; the
var mapStoreOps = map[string]string{
	shapeReader:       "Get",
	shapeWriter:       "Put",
	shapeAggregator:   "Count",
	shapeStreamReader: "List",
}

// KeyedStoreOp returns the [engine/model/ref.KeyedStore] method a shape
// delegates to — the keyed-put oracle, chosen where the interface's writer
// takes the key beside the value.
//
// A plain writer has no row: a keyed store cannot place a value whose key is
// not an argument, so a writer-shaped method under this oracle runs inert
// unless a mixin says what it is — see [KeyedStoreMixinOp].
func KeyedStoreOp(shape string) (string, bool) {
	op, ok := keyedStoreOps[shape]
	return op, ok
}

// KeyedStoreShapes returns every shape the keyed oracle models, sorted, for
// the census that holds each value to a method the oracle declares.
func KeyedStoreShapes() []string {
	out := make([]string, 0, len(keyedStoreOps))
	for s := range keyedStoreOps {
		out = append(out, s)
	}
	slices.Sort(out)
	return out
}

// KeyedStoreMixinOp returns the oracle method a mixin assigns to its carrier,
// where the shape alone says too little.
//
// The one row is the delete: Delete(ctx, k) is writer-shaped — one
// non-context parameter — and nothing in the signature distinguishes it from
// a put. The deleteremoves mixin is what states the semantics, and the stamp
// outranks the shape the way a partner reference does.
func KeyedStoreMixinOp(mixin string) (string, bool) {
	op, ok := keyedStoreMixinOps[mixin]
	return op, ok
}

// The keyed oracle's tables, method names spelled here for the reason
// mapStoreOps spells its own.
//
//nolint:gochecknoglobals,goconst // lookup tables, read-only after init; the
var (
	keyedStoreOps = map[string]string{
		shapeReader:          "Get",
		shapeCompositeWriter: "Put",
		shapeAggregator:      "Count",
	}
	keyedStoreMixinOps = map[string]string{
		mixinDeleteRemoves: "Delete",
	}
)
