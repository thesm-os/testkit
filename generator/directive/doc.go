// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package directive holds the canonical //testkit: directive vocabulary.
// It owns:
//
//   - the [Registry] of known directives and their [Descriptor]s
//   - the [ConsumerRegistry] mapping directive names to the generators
//     that enrich against them (inverted from the legacy
//     "Descriptor.Generators []string" model)
//   - composition rules ([ValidateComposition]) for conflict / required-
//     pair / redundancy detection
//
// Directive parsing — extracting //testkit: lines from doc comments —
// lives in the parent package as [generator.Directive] and the
// (unexported) parseDirectivesFromDoc, because [generator.MethodInfo]
// and friends carry slices of Directive and a circular import would
// otherwise be required.
//
// Generators register themselves as consumers at init time:
//
//	func init() {
//	    directive.RegisterConsumer("errors", "stub", enrichErrors)
//	    directive.RegisterConsumer("integration-only", "stub", enrichIntegrationOnly)
//	}
//
// The descriptor table is therefore the single source of truth for
// directive metadata; the consumer relationship is computed from
// registrations rather than hand-maintained alongside the descriptors.
package directive
