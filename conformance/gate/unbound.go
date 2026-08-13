// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

// The chokepoints several laws share, spelled once.
const (
	historyDebt = "waits on the history option — a subject that cannot report " +
		"its own transactions cannot be asked about isolation"
	drainDebt = "waits on the drain option — the subscription hands back a live " +
		"channel no generated closure drains honestly"
	twoPhase    = "waits on a begin that returns the transaction handle commit and rollback thread"
	pageDebt    = "waits on a page-shaped reader — the pagination fixture's keyed read has no cursor to resume from"
	comparator  = "waits on the supplied comparator its manifest names, which no generated value can stand in for"
	sessionDebt = "waits on a write that answers its stored state — the trace records " +
		"what was sent, never the version the store assigned, and an upserter " +
		"shape (ctx, V) (V, error) is the detector vocabulary upstream adds"
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
	// Waiting on a consumer-supplied option no generated value can stand in
	// for — an equality, an order, a projection only a domain knows. Each
	// reason names the option the generated header points at.
	"AUTO-CAUSAL-ORDERING":                  "waits on the happens-before option — causality is a relation over operations no stamp can state",
	"AUTO-EVENTUAL-CONVERGENCE":             "waits on the merge option — the replica lattice's join is the consumer's algebra",
	"AUTO-LEASE-RELEASED-ON-CANCEL":         "waits on the free option, which no generated value can stand in for",
	"AUTO-POOL-BALANCED":                    "waits on the stats option, which no generated value can stand in for",
	"AUTO-POOL-LEAK-FREE":                   "waits on the balanced option, which no generated value can stand in for",
	"AUTO-REPLAY-CAUSAL-ORDERING":           "waits on the entry-id and depends-on options — a dependency graph over entries is the consumer's causality",
	"AUTO-SNAPSHOT-ISOLATION-G0":            historyDebt,
	"AUTO-SNAPSHOT-ISOLATION-G1":            historyDebt,
	"AUTO-SNAPSHOT-ISOLATION-G2":            historyDebt,
	"AUTO-STREAM-OVER-MATCH":                comparator,
	"AUTO-STREAM-PERMUTATION":               comparator,
	"AUTO-STREAM-STABLE-ORDER":              "waits on the supplied Less its manifest names, which no generated value can stand in for",
	"AUTO-TRANSACTION-NO-MID-TX-VISIBILITY": "waits on the tx-put option — both mid-transaction writes live on a handle the roles do not reach",
	"AUTO-WATCHER-RETURNS-ON-CHANGE":        "waits on the next and stop options — both live on the handle Watch returns, which the roles do not reach",

	// Waiting on a role shape no fixture can declare within the contract's
	// current vocabulary: a callable-taking call, a handle-returning begin.
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

	// Waiting on a live subscription drain: the publisher's subscribe hands
	// back a channel, and a generated closure draining one honestly needs
	// the delivery design the twin-floor programme scopes.
	"AUTO-PUBLISHER-AT-LEAST-ONCE": drainDebt,
	"AUTO-PUBLISHER-AT-MOST-ONCE":  drainDebt,
	"AUTO-PUBLISHER-DELIVERS":      drainDebt,
	"AUTO-PUBLISHER-EXACTLY-ONCE":  drainDebt,

	// Waiting on an observable lifecycle carrier: the corpus's only
	// idempotent teardown is a Close-only interface, and the law reads
	// state before and after the second close — through a method the
	// fixture does not have. The claim it could keep without one is
	// close-idempotence alone, which is the cursor family's law.
	"AUTO-IDEMPOTENT-LIFECYCLE": "waits on an observable lifecycle carrier — the only idempotent teardown here is Close-only, and the law observes state across the second close",

	// Waiting on the append-recording hook.
	"AUTO-APPEND-ONLY-NO-DROPS": "waits on an append-recording history hook the runner does not offer",

	// The read-ordering half of the session family binds — monotonicreads
	// runs per client over the concurrent leg's trace. The write-ordering
	// three still cannot see the version a write was assigned: a writer
	// answering only an error hides it from the trace, and the shape that
	// surfaces it is a detector eidos does not yet draw.
	"AUTO-MONOTONIC-WRITES":    sessionDebt,
	"AUTO-READ-YOUR-WRITES":    sessionDebt,
	"AUTO-WRITES-FOLLOW-READS": sessionDebt,
}
