// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package generator is the elite-level rebuild of testkit's code generation
// infrastructure. It supersedes gen/ once the shadow build is complete; until
// then, both packages coexist and the CLI dispatches to gen/ for the ready
// generators.
//
// # Architecture
//
// The package is organized as a one-way dependency graph:
//
//	generator/                       (foundation: Generator, Pipeline, Loader,
//	                                  rendering, types)
//	├── shape/                       Detector registry + canonical shapes
//	├── directive/                   Registry, descriptors, parsing,
//	                                  composition rules
//	├── spec/                        Shared analyzer used by suite/, bench/,
//	                                  model/
//	└── stub/, builder/, sentinel/, enum/, suite/, bench/, model/
//	                                  Per-generator packages — each ~25 lines
//	                                  of pipeline configuration plus
//	                                  generator-specific data, enrichment,
//	                                  and templates.
//
// No generator imports another generator. Cross-cutting analysis lives in
// generator/spec/ — never inside one of its consumers.
//
// # Pipeline as template-method
//
// Every generator wires its analyze, enrich, and render steps into the
// shared [Pipeline]. The pipeline owns type validation, directive
// validation, enrichment dispatch, composition validation, template parsing,
// source attribution, header construction, and rendering — extracted from
// the seven hand-rolled Generate methods in gen/. Adding a cross-cutting
// concern (instrumentation, dry-run mode, debug dumps) is a one-place
// change.
//
// # Extension points
//
// Three registries make the system extensible without modification:
//
//   - [Registry] (this package) — generators register themselves by name.
//   - [shape.Registry] — detectors register with priority; adding a new
//     shape is a new file plus a registration call.
//   - [directive.Registry] — directives register descriptors;
//     [directive.RegisterConsumer] inverts the consumer relationship so
//     generators declare which directives they enrich, rather than
//     descriptors hard-coding which generators consume them.
//
// # Stability
//
// Pre-1.0. Public API may change between minor versions until the cutover
// from gen/ is complete. After cutover, additive changes only.
package generator
