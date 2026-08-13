// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

// UnboundLaws is the debt register the assertion gate carries: model-owned
// laws the corpus's stamps select, whose engine implementations ship tested,
// and which no fixture binds — each with the chokepoint that holds it. The
// generated-suite audit (docs/internal/generated-suite-audit.md) proved the
// class by deleting a fixture's whole claim and watching the corpus stay
// green; this register is that finding turned into a contract.
//
// The register only shrinks. The gate holds it two ways: a law selected by
// the corpus and bound nowhere must appear here or the build is red, and an
// entry that starts binding must be deleted or the build is red — debt
// recorded, never laundered. Quarantined laws are not listed: an unsound
// conduct cannot bind by design, and the conduct census already carries it.
//
//nolint:gochecknoglobals // a census table, read-only, test-facing.
var UnboundLaws = map[string]string{
	// Waiting on the multi-replica role closures: Sync and Settle range
	// over replicas no transcribed shape spells.
	"AUTO-EVENTUAL-CONVERGENCE": "waits on the multi-replica role closures — Sync and Settle range over replicas no transcribed shape spells",

	// Waiting on a fixture that can honestly stamp the pair: the rule needs
	// chain beside causal, and no corpus interface carries both.
	"AUTO-REPLAY-CAUSAL-ORDERING": "waits on a fixture stamping chain beside causal — the rule needs both, and no corpus interface carries the pair",

	// The member scope reaches the handle now; what remains is the carrier:
	// the law's TxPut and TxRollback take the handle alone, so the handle
	// must carry its own transaction, and no fixture declares one that does.
	"AUTO-TRANSACTION-NO-MID-TX-VISIBILITY": "waits on a handle that carries its transaction — the law's mid-tx write and rollback take the handle alone, and no fixture declares members on the handle Begin answers",
}
