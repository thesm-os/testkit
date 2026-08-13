// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

// UnboundLaws is the debt register the assertion gate carries: model-owned
// laws the corpus's stamps select, whose engine implementations ship tested,
// and which no fixture binds — each with the chokepoint that holds it. The
// generated-suite audit (docs/superpowers/model-audit.md) proved the
// class by deleting a fixture's whole claim and watching the corpus stay
// green; this register is that finding turned into a contract.
//
// The register only shrinks. The gate holds it two ways: a law selected by
// the corpus and bound nowhere must appear here or the build is red, and an
// entry that starts binding must be deleted or the build is red — debt
// recorded, never laundered. Quarantined laws are not listed: an unsound
// conduct cannot bind by design, and the conduct census already carries it.
//
// The register is empty: every law the corpus's stamps select is bound in
// at least one fixture, or consumer-payable through a generated door a
// fixture arms. The declaration stays because the gate's contract does — a
// law that stops binding must land here with its chokepoint or the build is
// red, and the empty literal is the waterline regression is measured
// against.
//
//nolint:gochecknoglobals // a census table, read-only, test-facing.
var UnboundLaws = map[string]string{}
