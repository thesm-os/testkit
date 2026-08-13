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
	// Waiting on a piece beside the generated door: the supplied-option
	// surface exists for both, and what still refuses is the classifier's
	// missing ordering stamp and the multi-replica role closures.
	"AUTO-CAUSAL-ORDERING":      "waits on a version= member for its classifier — the causal mixin declares none, and per-client ops need the ordering stamp",
	"AUTO-EVENTUAL-CONVERGENCE": "waits on the multi-replica role closures — Sync and Settle range over replicas no transcribed shape spells",

	// Waiting on a fixture that can honestly stamp the pair: the rule needs
	// chain beside causal, and no corpus interface carries both.
	"AUTO-REPLAY-CAUSAL-ORDERING": "waits on a fixture stamping chain beside causal — the rule needs both, and no corpus interface carries the pair",

	"AUTO-TRANSACTION-NO-MID-TX-VISIBILITY": "waits on the handle-member scope — the mid-transaction write is a method of the handle Begin answers, which neither resolver scope reaches; upstream owns the scope",
	"AUTO-WATCHER-RETURNS-ON-CHANGE":        "waits on the handle-member scope — Next and Stop are methods of the handle Watch answers, which neither resolver scope reaches; upstream owns the scope",

	// Waiting on version-coherent draws: a compare-and-swap only wins when
	// its expected version matches the cell's, and a static pool cannot read
	// the cell at draw time. The VersionedCell oracle now guards the
	// differential; the law's paired-attempt shape stays out of reach until
	// a draw can ask the cell where it stands.
	"AUTO-CAS-ATOMIC-ONE-WINNER": "waits on version-coherent draws — a static pool cannot read the cell's version at draw time",

	// Waiting on the append-recording hook.
	"AUTO-APPEND-ONLY-NO-DROPS": "waits on an append-recording history hook the runner does not offer",
}
