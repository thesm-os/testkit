// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package spec holds the shared analysis model consumed by every
// interface-conformance generator (stub, suite, bench, model).
//
// The package owns:
//
//   - [Data] — the top-level analysis result for one interface
//   - [Method] — per-method analysis (signature, shape, directive
//     enrichment, mixin emissions)
//   - typed [Get] / [Set] attachment helpers for the per-method
//     directive payload map
//
// # Population stages
//
// [Data] is built in stages by the Phase 4 [Analyze] function (lands
// later — Phase 1.8 ships only the types):
//
//  1. Loader produces [generator.InterfaceInfo].
//  2. spec walks each method, runs [shape.Classify], populates [Method.Shape].
//  3. The directive consumer pass populates [Method.Attachments]
//     with enrichment payloads — `errors`, `sample`, `deprecated`, etc.
//  4. The directive emitter pass populates [Method.Attachments]
//     with mixin emissions — `atomic`, `bounded`, `roundtrip`, etc.
//
// Both passes share one map keyed by directive name. The directive
// [Category] (Mixin vs Enrichment) is queryable via the directive
// registry for code that needs to distinguish them.
//
// Generators consume the resulting [Data] read-only.
//
// # Attachment map
//
// [Method.Attachments] is intentionally typed as `map[string]any` so
// the spec package does not need to know every directive's payload
// type. Each consumer/emitter declares its own payload type and uses
// [Get] / [Set] for type-safe access:
//
//	spec.Set(&method.Attachments, directive.Errors, errorsenrich.Payload{...})
//	payload, ok := spec.Get[errorsenrich.Payload](method.Attachments, directive.Errors)
package spec
