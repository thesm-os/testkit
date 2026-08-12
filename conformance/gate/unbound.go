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
	// The bindings table has rows for 13 of 83 laws; everything below waits
	// on its instantiation row — type arguments after the subject — plus,
	// for many, a law-field template its manifest names and no renderer
	// spells yet. The engine side of every one ships and is unit-tested.
	"AUTO-AGGREGATOR-BOUNDED":               rowDebt,
	"AUTO-APPEND-ONLY-GROWS":                rowDebt,
	"AUTO-APPEND-ONLY-NO-DROPS":             rowDebt,
	"AUTO-APPENDER-MONOTONIC-OFFSETS":       rowDebt,
	"AUTO-ASSOCIATIVE":                      rowDebt,
	"AUTO-ATOMIC-WRITE":                     rowDebt,
	"AUTO-CAS-ATOMIC-ONE-WINNER":            rowDebt,
	"AUTO-CAUSAL-ORDERING":                  rowDebt,
	"AUTO-COMMUTATIVE-WRITE":                rowDebt,
	"AUTO-CONSERVATIVE":                     rowDebt,
	"AUTO-COUNT-EQUALS-REFERENCE":           rowDebt,
	"AUTO-CRDT-MERGE":                       rowDebt,
	"AUTO-DEADLINE-RESPECTING":              rowDebt,
	"AUTO-EVENTUAL-CONVERGENCE":             rowDebt,
	"AUTO-HASH-CHAIN-INTEGRITY-VERIFY":      rowDebt,
	"AUTO-IDEMPOTENT-WRITE":                 rowDebt,
	"AUTO-INJECTION-SAFE":                   rowDebt,
	"AUTO-LEAK-FREE":                        rowDebt,
	"AUTO-LEASE-DOUBLE-ACQUIRE-BLOCKS":      rowDebt,
	"AUTO-LEASE-RELEASED-ON-CANCEL":         rowDebt,
	"AUTO-LIFECYCLE-RESPECTS-CONTEXT":       rowDebt,
	"AUTO-LOSSY-ROUNDTRIP":                  rowDebt,
	"AUTO-MONOTONIC-NON-DECREASING":         rowDebt,
	"AUTO-MONOTONIC-READS":                  rowDebt,
	"AUTO-MONOTONIC-WRITES":                 rowDebt,
	"AUTO-PAGINATOR-NO-DUPLICATES":          rowDebt,
	"AUTO-PAGINATOR-RESUMABLE":              rowDebt,
	"AUTO-PERSISTER-RETRIEVABLE":            rowDebt,
	"AUTO-POISON-IDEMPOTENT-READ":           rowDebt,
	"AUTO-POISON-NIL-ON-FRESH":              rowDebt,
	"AUTO-POOL-BALANCED":                    rowDebt,
	"AUTO-POOL-LEAK-FREE":                   rowDebt,
	"AUTO-PREDICATE-CONSISTENT":             rowDebt,
	"AUTO-PUBLISHER-AT-LEAST-ONCE":          rowDebt,
	"AUTO-PUBLISHER-AT-MOST-ONCE":           rowDebt,
	"AUTO-PUBLISHER-DELIVERS":               rowDebt,
	"AUTO-PUBLISHER-EXACTLY-ONCE":           rowDebt,
	"AUTO-PURE-DETERMINISTIC":               rowDebt,
	"AUTO-READ-YOUR-WRITES":                 rowDebt,
	"AUTO-REPLAY-CAUSAL-ORDERING":           rowDebt,
	"AUTO-REPLAY-DETERMINISTIC":             rowDebt,
	"AUTO-ROUNDTRIP":                        rowDebt,
	"AUTO-SAGA-FULL-COMPENSATION":           rowDebt,
	"AUTO-SINGLEFLIGHT-COALESCES":           rowDebt,
	"AUTO-SNAPSHOT-ISOLATION-G0":            rowDebt,
	"AUTO-SNAPSHOT-ISOLATION-G1":            rowDebt,
	"AUTO-SNAPSHOT-ISOLATION-G2":            rowDebt,
	"AUTO-STREAM-REFLECTS-MUTATIONS":        rowDebt,
	"AUTO-TOTAL-OVER":                       rowDebt,
	"AUTO-TRANSACTION-NO-MID-TX-VISIBILITY": rowDebt,
	"AUTO-TRANSACTION-ROLLBACK":             rowDebt,
	"AUTO-TWO-PHASE-MUTEX":                  rowDebt,
	"AUTO-TWO-PHASE-ROLLBACK-AFTER-COMMIT":  rowDebt,
	"AUTO-UPDATER-REPLACES":                 rowDebt,
	"AUTO-UPSERTER-IDEMPOTENT":              rowDebt,

	"AUTO-WINDOWED": rowDebt,

	"AUTO-WATCHER-RETURNS-ON-CHANGE": rowDebt,

	"AUTO-WRITES-FOLLOW-READS": rowDebt,

	"AUTO-VALID-TRANSITION": rowDebt,

	"AUTO-XSS-SAFE": rowDebt,

	// These five have instantiation rows and stall one wall later: their
	// manifests name a supplied comparator or a handle the generator only
	// knows how to fill with the key projection.
	"AUTO-STREAM-COMPLETION":   "waits on the KindHandle generalization — its manifest's handle is not the key projection",
	"AUTO-STREAM-REENTRANT":    "waits on the KindHandle generalization — its manifest's handle is not the key projection",
	"AUTO-STREAM-OVER-MATCH":   "waits on the supplied comparator its manifest names, which no generated value can stand in for",
	"AUTO-STREAM-PERMUTATION":  "waits on the supplied comparator its manifest names, which no generated value can stand in for",
	"AUTO-STREAM-STABLE-ORDER": "waits on the supplied Less its manifest names, which no generated value can stand in for",

	// Has a row; its one fixture is a bare reader, and the row instantiates
	// at a value type nothing there draws.
	"AUTO-CACHEABLE": "waits on a cacheable fixture whose shape draws the value type its row instantiates",
}

// rowDebt is the register's dominant chokepoint, spelled once.
const rowDebt = "waits on an instantiation row in the bindings table naming its type arguments"
