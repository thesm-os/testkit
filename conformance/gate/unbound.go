// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

// The chokepoints several laws share, spelled once.
const (
	twoPhase = "waits on a begin that returns the transaction handle commit and rollback thread"
	pageDebt = "waits on a page-shaped reader — the pagination fixture's keyed read has no cursor to resume from"
)

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

	// Waiting on fixture shapes and their closure derivations — ours, not
	// vocabulary: the contract roles are name-anchored, so a
	// handle-answering begin, a callable-taking call and a cursor-shaped
	// page read all resolve today; what is missing is a fixture that
	// declares each shape and the generator arms that spell its closures.
	"AUTO-SAGA-FULL-COMPENSATION":          "waits on a run the mirrored pair can repeat and an observation of compensated state, and the saga's step role offers neither",
	"AUTO-SINGLEFLIGHT-COALESCES":          "waits on a compute-taking call shape the singleflight fixture's role method does not declare",
	"AUTO-TRANSACTION-ROLLBACK":            "waits on a run role that accepts the failing body its manifest threads through the transaction",
	"AUTO-TWO-PHASE-MUTEX":                 twoPhase,
	"AUTO-TWO-PHASE-ROLLBACK-AFTER-COMMIT": twoPhase,
	"AUTO-PAGINATOR-NO-DUPLICATES":         pageDebt,
	"AUTO-PAGINATOR-RESUMABLE":             pageDebt,
	"AUTO-ATOMIC-WRITE":                    "waits on a single-input atomic write beside a whole-state observation — the fixture pairs several inputs with a two-field read",

	// Waiting on version-coherent draws: a compare-and-swap only wins when
	// its expected version matches the cell's, and a static pool cannot read
	// the cell at draw time. The VersionedCell oracle now guards the
	// differential; the law's paired-attempt shape stays out of reach until
	// a draw can ask the cell where it stands.
	"AUTO-CAS-ATOMIC-ONE-WINNER": "waits on version-coherent draws — a static pool cannot read the cell's version at draw time",

	// Waiting on an observable lifecycle carrier: the corpus's only
	// idempotent teardown is a Close-only interface, and the law reads
	// state before and after the second close — through a method the
	// fixture does not have. The claim it could keep without one is
	// close-idempotence alone, which is the cursor family's law.
	"AUTO-IDEMPOTENT-LIFECYCLE": "waits on an observable lifecycle carrier — the only idempotent teardown here is Close-only, and the law observes state across the second close",

	// Waiting on the append-recording hook.
	"AUTO-APPEND-ONLY-NO-DROPS": "waits on an append-recording history hook the runner does not offer",
}
