// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package model provides property-based state-machine testing for Go
// interfaces, driven by shape classification.
//
// Built on [pgregory.net/rapid] for generation and shrinking, the
// framework synthesizes state machines from the same shape detection
// that powers the suite and bench generators. Consumers get
// shape-derived algebraic invariants (read-after-write,
// delete-removes-value, count-equals-keys) with zero configuration.
//
// Three tiers of adoption:
//
//   - Tier 0: shape-derived auto-invariants only. Zero consumer code
//     beyond the entry point.
//   - Tier 1: consumer-supplied reference model. Framework checks
//     SUT ≡ reference after every command.
//   - Tier 2: custom [Law] implementations for domain-specific
//     invariants.
//
// The [Runner] is generic over the interface under test. Composition
// is done at construction time — a composed reference that implements
// multiple interfaces is still a single T from the runner's
// perspective.
package model
