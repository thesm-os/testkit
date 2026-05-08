// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec

import (
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/shape"
)

// Data is the analyzed view of one interface, consumed by stub,
// suite, bench, and model generators. Populated by [Analyze] (Phase 4).
//
// Each generator typically wraps Data via composition rather than
// extending it:
//
//	type benchData struct {
//	    *spec.Data
//	    BenchSpecificField string
//	}
//
// Embedding keeps the shared model immutable from the generator's
// view; generator-specific state lives on the wrapper.
type Data struct {
	// Package is the loaded source package. Generators query the
	// package for type lookups, position information, and import
	// resolution.
	Package *generator.Package

	// Interface is the fully populated [generator.InterfaceInfo] —
	// name, type parameters, methods (including embedded), doc,
	// directives at the type level, and source position.
	Interface generator.InterfaceInfo

	// Methods is one [Method] per interface method, in the order
	// returned by the loader (sorted alphabetically). Holds shape
	// classification plus per-method directive payloads.
	Methods []Method

	// Tracker is the [generator.ImportTracker] used to render type
	// names in generated output. Owned by Data; methods and
	// templates reference it via this single instance to keep the
	// import alias map consistent across all rendering.
	Tracker *generator.ImportTracker

	// Args carries the original CLI argv (e.g. ["Store"] or
	// ["Cache", "--variant=fast"]) — used by [generator.Header] for
	// source attribution lines and by some enrichers that consult
	// flags.
	Args []string
}

// Method is the analyzed view of one interface method. Embeds
// [generator.MethodInfo] so generators can promote Name, Signature,
// Doc, Directives, and Pos directly.
//
// The Attachments map carries directive-driven payloads populated by
// the consumer pass (enrichment) and the emitter pass (mixin
// emissions). Both passes use the directive name as the key, so any
// given method exposes a flat namespace of `directive name → payload`.
//
// The map is nil for fresh [Method] values; use [Set] (which lazily
// allocates) to write.
type Method struct {
	// MethodInfo is the underlying loader output. Embedded for
	// promotion: m.Name, m.Signature, m.Doc, m.Directives, m.Pos.
	generator.MethodInfo

	// Shape is the detected shape from [shape.Classify]. Set during
	// the analysis pass before enrichment runs.
	Shape shape.Info

	// Attachments carries per-directive payloads keyed by directive
	// name. Both enrichers (errors → fault-helper data, sample →
	// bench inputs) and mixin emitters (atomic → assertion suite,
	// roundtrip → assertion suite) write into this single map.
	//
	// The directive [Category] (Mixin vs Enrichment vs ...) is
	// queryable via the directive registry when callers need to
	// distinguish emit-time vs read-time payloads.
	//
	// Generators read via [Get]; consumers and emitters write via
	// [Set].
	Attachments map[string]any
}
