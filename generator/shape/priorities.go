// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// Priority constants for the shipped detectors. The cascade is the
// single source of truth for detector ordering; each detector's
// `Priority()` method returns its constant from this file. The
// registry sorts descending and dispatches first-match-wins.
//
// Higher values fire first. Gaps in the numbering leave room for
// future detectors without renumbering existing ones.
//
// Audit invariants (enforced by [TestPriorityUniqueness] in tests):
//
//   - Every priority is unique
//   - Every detector has a priority constant in this file
//   - The constant order matches the dispatch order
//
// Adding a new detector: pick a number in the appropriate band that
// doesn't collide with existing priorities, declare the constant
// here, register the detector in [defaultDetectors].
const (
	// Stream-typed returns claim outright — iter.Seq[T] is
	// unambiguous regardless of other shape signals.
	PriorityStreamReader = 1000

	// Variadic-keyed batch reads.
	PriorityBatchReader = 950

	// Interface-typed non-ctx parameter (io.Reader, io.Writer, etc).
	PriorityStreamConsumer = 900

	// Bool-discriminated lookups (3 results, last bool).
	PriorityLookup = 850

	// Bool-discriminated single-value reads (2 results, last bool).
	PriorityReaderWithBool = 840

	// Exact-match no-arg shapes claim before generic patterns.
	PriorityPoisonAccessor = 830 // () error
	PriorityPredicate      = 820 // () bool
	PriorityVoidLifecycle  = 810 // () or (ctx)
	PriorityPure           = 800 // () T

	// Multi-arg ctx-respecting writes.
	PriorityMultiArgWriter = 750

	// Two-key writes.
	PriorityCompositeWriter = 700

	// Multi-result reads.
	PriorityMultiReader     = 650
	PriorityMultiAggregator = 600

	// Directive-elevated single-key shapes claim before generic
	// signature catches them.
	PriorityDeleter = 550

	// Single-key error-only writes.
	PriorityWriter = 500

	// Pointer-typed result claims before generic Reader or
	// ReaderNoError.
	PriorityPointerReader = 450

	// Single-key reads.
	PriorityReader        = 420
	PriorityReaderNoError = 400

	// Ctx-only or error-only aggregations.
	PriorityAggregator = 350

	// Auto-detected void-return mutations.
	PriorityMutator = 300

	// Ctx + error-only fallbacks last.
	PriorityLifecycle = 200

	// Composite-tier band: multi-method shapes claim before any
	// signature-tier or contract-tier detection. Saga at the top —
	// directive-required, intercepts before the auto-detected ones.
	PriorityCompositeSaga     = 2050
	PriorityCompositeTwoPhase = 2030
	PriorityCompositeCursor   = 2010
	PriorityCompositePool     = 2000

	// Contract-tier band: directive-promoted or structurally-derived
	// single-method contracts. Each fires after composite-tier and
	// before signature-tier, allowing a directive on a Reader/Writer
	// to elevate it to a contract-tier shape with stronger invariants.
	PriorityContractTransactionFunc = 1590
	PriorityContractGetOrCompute    = 1580
	PriorityContractCAS             = 1570
	PriorityContractPaginator       = 1560
	PriorityContractAcquireLease    = 1550
	PriorityContractPersister       = 1540
	PriorityContractUpdater         = 1530
	PriorityContractUpserter        = 1520
	PriorityContractAppender        = 1510
	PriorityContractWatcher         = 1505
	PriorityContractPublisher       = 1503
	PriorityContractSubscriber      = 1502
)
