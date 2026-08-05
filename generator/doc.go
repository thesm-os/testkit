// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package generator is the root of testkit's code generators.
//
// Each generator is one package under this module — [stub], [suite], [bench],
// and the rest of the catalogue — and each implements the eidos plugin roles it
// needs: a generator plugin always, plus directive, template, or capability
// providers where the generator has a vocabulary of its own.
//
// # Boundary
//
// testkit supplies generators and annotator configuration. Everything else —
// frontend, intermediate representation, typed metadata, slot ordering,
// determinism, caching, and the sink — comes from eidos. testkit consumes
// eidos's annotator plugin and none of its generators; see docs/adr/0004.
//
// Classification is therefore configured rather than implemented. A generator
// reads the three axes eidos's shape annotator stamps — detector for the
// signature, contract for the multi-callable protocol and role, mixin for the
// declared guarantees — and decides what to emit from those. It does not
// inspect Go types directly.
//
// # Hazards
//
// Generators run inside eidos's pipeline and inherit its concurrency model:
// plugins in the same priority bucket may run in parallel, so a generator must
// not hold state across Generate calls. Reads go through the per-plugin
// store.Reader rather than the store directly, because the captured reads
// become the plugin's cache key — bypassing it produces stale output that
// looks correct.
//
// Determinism is a hard requirement, not an aspiration: identical inputs must
// produce byte-identical output across runs and machines. Map iteration, time,
// and randomness are all sources of failure here.
package generator
