// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package interfaces holds the production-grade interface fixtures
// every interface-conformance generator (stub, suite, bench, model)
// consumes for shape-coverage testing.
//
// Three interfaces, each with a paired in-memory companion:
//
//   - [AllShapes] / [InMemoryAllShapes]: one method per signature-tier
//     shape from generator/shape's catalog (Reader through Unknown).
//     Covers all 22 detectable shapes plus the Unknown fall-through.
//   - [Directives] / [InMemoryDirectives]: full stub-relevant
//     directive vocabulary (errors, integration-only, deprecated,
//     retry-succeeds-on-attempt, partition, order-after, wrapped-via).
//   - The generics counterparts live in testdata/generics/ifaces.go
//     so generic-type fixtures stay grouped with the rest of the
//     generic surface.
//
// Companion impls are deliberately minimal — map-backed where
// stateful, return-zero where stateless — enough to thread
// DelegateTo verification without re-implementing production logic.
package interfaces
