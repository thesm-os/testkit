// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec

import (
	"go/types"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/shape"
)

// Data is the analyzed view of one interface, consumed by stub,
// suite, bench, and model generators. Populated by [Analyze].
//
// Each generator wraps Data via composition rather than extending
// it:
//
//	type Data struct {
//	    *spec.Data
//	    BenchSpecificField string
//	}
//
// Embedding keeps the shared model immutable from the generator's
// view; generator-specific state lives on the wrapper.
type Data struct {
	// PackageName is the package declaration the output file uses,
	// computed via [generator.DerivePackageName]. The test-view
	// renderer reshapes this via [generator.BuildTestFileInfo] when
	// emitting an external _test file.
	PackageName string

	// Imports is the resolved import set for the output file,
	// computed from the [Tracker] after analysis (which records
	// every package referenced by the rendered method signatures).
	Imports []generator.Import

	// ImplImportPath is the import path of the impl file's
	// package, computed via [generator.BuildOutputCtx]. The
	// test-view renderer uses it to derive the test file's
	// GenQualifier.
	ImplImportPath string

	// Package is the loaded source package. Generators query the
	// package for type lookups, position information, and import
	// resolution during enrichment.
	Package *generator.Package

	// Interface is the fully populated [generator.InterfaceInfo] —
	// name, type parameters, methods (including embedded), doc,
	// directives at the type level, and source position.
	Interface generator.InterfaceInfo

	// QualifiedType is the source-package-qualified interface
	// reference, suffixed with type-args for generics. Examples:
	//
	//	non-generic:        "basic.Store"
	//	generic Cache:      "generics.Cache[K, V]"
	QualifiedType string

	// TypeParamDecl renders the interface's type-parameter
	// declaration when generic, e.g. "[K comparable, V any]".
	// Empty for non-generic interfaces.
	TypeParamDecl string

	// TypeParamArgs renders just the type-parameter names for
	// instantiation, e.g. "[K, V]". Empty for non-generic.
	TypeParamArgs string

	// IsGeneric is true when [TypeParamDecl] is non-empty.
	IsGeneric bool

	// Methods is one [Method] per interface method, in the order
	// returned by the loader (sorted alphabetically). Holds shape
	// classification plus per-method directive payloads.
	Methods []Method

	// Tracker is the [generator.ImportTracker] used to render type
	// names in generated output. Owned by Data; methods and
	// templates reference it via this single instance to keep the
	// import alias map consistent across all rendering.
	Tracker *generator.ImportTracker

	// Loader is the [generator.Loader] used by consumers that need
	// to resolve symbols in packages other than [Package] — e.g.
	// the sample directive consumer when a directive arg names a
	// fully-qualified function in a fixtures package. Owned by Data
	// so cache hits accumulate across all consumers in one Enrich
	// pass.
	Loader *generator.Loader

	// Args carries the original CLI argv (e.g. ["Store"] or
	// ["Cache", "--variant=fast"]) — used for source attribution
	// and by some enrichers that consult flags.
	Args []string

	// Directives are the //testkit: package-level directives
	// consumed by spec, pre-rendered as source lines for the header
	// partial. Empty when none apply.
	Directives []string
}

// Method is the analyzed view of one interface method. Embeds
// [generator.MethodInfo] so generators can promote Name, Signature,
// Doc, Directives, and Pos directly.
//
// The Attachments map carries directive-driven payloads populated
// by the consumer pass (enrichment) and the emitter pass (mixin
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

// NonCtxParamCount returns the number of parameters excluding a
// leading [context.Context]. Useful to consumers whose arg-shape
// rule is "one per non-ctx param" (sample, hooks, validates).
func (m *Method) NonCtxParamCount() int {
	n := m.Signature.Params().Len()
	if m.HasContext() {
		n--
	}
	return n
}

// NonCtxParamAt returns the type of the i-th non-ctx parameter.
// Pairs with [NonCtxParamCount] for consumers iterating directive
// args alongside the corresponding parameter types.
func (m *Method) NonCtxParamAt(i int) types.Type {
	offset := 0
	if m.HasContext() {
		offset = 1
	}
	return m.Signature.Params().At(offset + i).Type()
}
