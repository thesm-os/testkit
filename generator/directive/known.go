// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive

import "sync"

// Directive name constants. Use these in code that needs to compare
// directive names — the typed string keeps refactors and renames safe.
const (
	// Error & return contract
	Errors     = "errors"
	WrappedVia = "wrapped-via"

	// Behavioral properties (mixin tier).
	Idempotent = "idempotent"
	Pure       = "pure"
	Cacheable  = "cacheable"
	Monotonic  = "monotonic"

	// Safety (mixin tier).
	Concurrent        = "concurrent"
	ConcurrentReaders = "concurrent-readers"
	NilSafe           = "nilsafe"
	Atomic            = "atomic"

	// Context & lifecycle.
	Ctx             = "ctx"
	Timeout         = "timeout"
	Deprecated      = "deprecated"
	Lease           = "lease"
	IntegrationOnly = "integration-only"

	// Performance.
	Allocs      = "allocs"
	Latency     = "latency"
	Percentiles = "percentiles"

	// Resilience.
	Retryable              = "retryable"
	RetrySucceedsOnAttempt = "retry-succeeds-on-attempt"

	// Causality & ordering.
	SideEffect = "sideeffect"
	OrderAfter = "order-after"

	// Isolation.
	Partition = "partition"

	// Input & validation.
	Validates = "validates"
	Bounded   = "bounded"

	// Properties & invariants.
	Invariant = "invariant"

	// Testing & observability.
	Fuzz  = "fuzz"
	Hooks = "hooks"
	Req   = "req"

	// Consistency.
	Eventually = "eventually"

	// Authorization.
	Scope = "scope"

	// Iteration.
	Pagination = "pagination"

	// Shape hints.
	Deleter    = "deleter"
	Mutator    = "mutator"
	NotMutator = "not-mutator"
	KeyField   = "keyfield"

	// Sample directive (smoke + bench input replacement).
	Sample = "sample"

	// Cross-method invariants. Each is a first-class directive; the
	// suite emits one assertion per directive at the carrier method's
	// t.Run block. Adding a new invariant is one new descriptor +
	// consumer + runtime helper + template — no umbrella `cross`
	// indirection, no closed set. The `Cross` directive remains for
	// future generic invariant declarations not yet shaped into a
	// per-invariant directive.
	Cross                   = "cross"
	ReadAfterWrite          = "read-after-write"
	DeleteRemoves           = "delete-removes"
	StreamReflectsMutations = "stream-reflects-mutations"
	LifecycleAfterClose     = "lifecycle-after-close"
	CRDTMerge               = "crdt-merge"

	// Sentinel cross-package non-overlap.
	SentinelNoOverlapWith = "sentinel-no-overlap-with"

	// Builder per-field default (`//testkit:default "value"`). Field-
	// scoped, consumed by the builder generator to seed Build() with
	// a non-zero literal when no Defaults factory exists.
	Default = "default"

	// Chain mode: append-only history with replay. The carrier method
	// is a writer-class that appends to a chain; consumers emit
	// hash-chain integrity, replay determinism, and partition-aware
	// causality laws.
	NonDeterministic = "nondeterministic"
	TimeAware        = "time-aware"
	Appends          = "appends"
	Replays          = "replays"
	Verifies         = "verifies"
	EntryID          = "entry-id"
	DependsOn        = "depends-on"
	Hash             = "hash"

	// Contract-tier promotions: shape promoters that lift signature-
	// tier shapes into invariant-carrying contracts (Persister,
	// CompareAndSwap, Appender, etc). Detected automatically from
	// interface structure; the directive asserts the contract for
	// codegen verification.
	Persister    = "persister"
	Updater      = "updater"
	Upserter     = "upserter"
	CAS          = "cas"
	Appender     = "appender"
	Watcher      = "watcher"
	Singleflight = "singleflight"
	Transaction  = "transaction"
	Acquire      = "acquire"
	Publisher    = "publisher"
	Subscribe    = "subscribe"

	// Composite-tier promotions: multi-method shapes spanning ≥2
	// interface methods. Saga is the only composite-tier shape that
	// requires the directive (auto-detection is too unreliable); the
	// others auto-detect with the directive available as override.
	Pool     = "pool"
	Cursor   = "cursor"
	TwoPhase = "two-phase"
	Saga     = "saga"

	// Mixins: orthogonal invariants that compose with any base shape.
	// atomic / idempotent / monotonic / pure / nilsafe / cacheable /
	// concurrent / bounded are registered above; the entries below
	// are additional mixin directives.
	Commutative         = "commutative"
	Associative         = "associative"
	Permutation         = "permutation"
	Windowed            = "windowed"
	CASAtomic           = "cas-atomic"
	PointInTime         = "point-in-time"
	LeakFree            = "leak-free"
	TamperEvident       = "tamper-evident"
	ConstantTime        = "constant-time"
	Delivery            = "delivery"
	DefaultOnError      = "default-on-error"
	ValidTransitionOnly = "valid-transition-only"
	OverMatchAcceptable = "over-match-acceptable"
	XSSSafe             = "xss-safe"
	InjectionSafe       = "injection-safe"
	TotalOver           = "total-over"
	PaginationResumable = "pagination-resumable"
	Roundtrip           = "roundtrip"
	LossyRoundtrip      = "lossy-roundtrip"
	StableOrder         = "stable-order"
	Sticky              = "sticky"
	Conservative        = "conservative"

	// Consistency-model selectors: per-interface or per-method
	// consistency assertion. linearizable, causal, snapshot-isolation,
	// and eventually are mutually exclusive at the interface level;
	// per-client guarantees compose with any of the others.
	Linearizable      = "linearizable"
	Causal            = "causal"
	SnapshotIsolation = "snapshot-isolation"
	ReadYourWrites    = "read-your-writes"
	MonotonicReads    = "monotonic-reads"
	MonotonicWrites   = "monotonic-writes"
	WritesFollowReads = "writes-follow-reads"

	// Action distribution & precondition. Adjusts the property-based
	// engine's command-distribution weight (bias), gates an action
	// behind a Go boolean expression (precondition), or marks a law
	// as expensive enough to fire on a sampled subset of iterations.
	Bias         = "bias"
	Precondition = "precondition"
	Expensive    = "expensive"

	// Fault-injection control: inject faults during the named method's
	// execution window rather than before/after, surfacing mid-
	// operation crash-safety bugs.
	FaultWindowOf = "fault-window-of"

	// Model-internal: reference primitive override, domain-typed
	// generator registration, and additional named TestClock
	// declarations.
	Reference = "reference"
	DomainGen = "domain-gen"
	Clock     = "clock"
)

// defaultDescriptors returns the canonical descriptor list, declared
// via the [New] builder. Categories, composition metadata, and arg
// schemas live on each descriptor — there is no parallel rules table.
//
// New directives append to this list; the [Registry] indexes them by
// name on registration.
func defaultDescriptors() []Descriptor {
	return []Descriptor{
		// Error & return contract.
		New(
			Errors,
			Describe("sentinel error returns"),
			InCategory(Enrichment),
			InPhase(Phase1),
			Arg("ErrName", ArgIdent, Required, Multi),
			Consumed("stub", "Fault<Sentinel>() helper per name"),
			Consumed(
				"suite",
				"AssertReturnsSentinel/AssertWriteRejectInvalid drives the per-shape sentinel-return assertion",
			),
		),
		New(WrappedVia,
			Describe("error wrapping discipline"),
			InCategory(Enrichment),
			InPhase(Phase1),
			Arg("ErrName", ArgIdent, Required),
			Requires(Errors),
			Consumed("stub", "Fault<Sentinel> helpers wrap via target"),
			Consumed("suite", "AssertWrappedVia subtest verifies error chain"),
		),

		// Behavioral properties (Mixin tier).
		New(Idempotent,
			Describe("repeated calls produce same result"),
			InCategory(Mixin),
			InPhase(Phase1),
			Consumed("suite", "AssertIdempotentSecondCall subtest"),
		),
		New(Pure,
			Describe("no side effects"),
			InCategory(Mixin),
			InPhase(Phase1),
			ConflictsWith(SideEffect, Monotonic),
			Implies(Idempotent),
			Consumed("suite", "AssertPureImplIndependent cross-impl agreement subtest"),
		),
		New(Cacheable,
			Describe("deterministic function of inputs"),
			InCategory(Mixin),
			InPhase(Phase3),
			Implies(Pure),
			Consumed("suite", "AssertCacheableRepeatedReads three-call equality subtest"),
		),
		New(Monotonic,
			Describe("ordered results"),
			InCategory(Mixin),
			InPhase(Phase3),
			Consumed("suite", "AssertMonotonicNonDecreasing subtest (cmp.Ordered required)"),
		),

		// Safety (Mixin tier).
		New(Concurrent,
			Describe("safe for parallel access"),
			InCategory(Mixin),
			InPhase(Phase1),
			ConflictsWith(ConcurrentReaders),
			Consumed("suite", "AssertConcurrentStrict 16×25 fanout subtest"),
		),
		New(ConcurrentReaders,
			Describe("parallel reads, serialised writes"),
			InCategory(Mixin),
			InPhase(Phase3),
			Consumed("suite", "AssertConcurrentReadersParallel 32-reader fanout subtest"),
		),
		New(NilSafe,
			Describe("zero/nil inputs do not panic"),
			InCategory(Mixin),
			InPhase(Phase1),
			Consumed("suite", "AssertNilSafeNoPanic subtest"),
		),
		New(Atomic,
			Describe("all-or-nothing"),
			InCategory(Mixin),
			InPhase(Phase3),
			Requires(Errors),
			Consumed("suite", "AssertAtomicNoTrace failure-path observation subtest"),
		),

		// Context & lifecycle.
		New(Ctx,
			Describe("respects context cancellation"),
			InCategory(Mixin),
			InPhase(Phase1),
		),
		New(Timeout,
			Describe("must complete within deadline"),
			InCategory(Enrichment),
			InPhase(Phase1),
			Arg("duration", ArgDuration, Required),
			Consumed("suite", "AssertTimeoutWithin deadline-bound subtest"),
		),
		New(Deprecated,
			Describe("method is deprecated"),
			InCategory(Enrichment),
			InPhase(Phase5),
			Arg("Replacement", ArgString, Required),
			Consumed("stub", "tb.Logf in dispatch + // Deprecated: doc comment"),
			Consumed("suite", "AssertDeprecatedSmoke subtest"),
		),
		New(Lease,
			Describe("acquires resource, must release via the named method"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("Release", ArgIdent, Required),
			Consumed("suite", "AssertLeaseAcquireRelease pair-method subtest"),
		),
		New(IntegrationOnly,
			Describe("opt out of stubbing"),
			InCategory(Enrichment),
			InPhase(Phase5),
			Consumed("stub", "skip dispatch (zero return, no record)"),
			Consumed("suite", "documented t.Skip in per-method block"),
		),

		// Performance.
		New(Allocs,
			Describe("allocation ceiling"),
			InCategory(Enrichment),
			InPhase(Phase2),
			Arg("N", ArgInt, Required),
		),
		New(Latency,
			Describe("latency ceiling"),
			InCategory(Enrichment),
			InPhase(Phase2),
			Arg("duration", ArgDuration, Required),
		),
		New(Percentiles,
			Describe("per-percentile latency ceilings"),
			InCategory(Enrichment),
			InPhase(Phase2),
			Arg("budget", ArgString, Required, Multi),
		),

		// Resilience.
		New(Retryable,
			Describe("safe to retry"),
			InCategory(Mixin),
			InPhase(Phase3),
			Consumed("stub", "required companion of retry-succeeds-on-attempt"),
		),
		New(RetrySucceedsOnAttempt,
			Describe("transient-failure recovery"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("N", ArgInt, Required),
			Requires(Retryable),
			Consumed("stub", "RetrySchedule(err) helper"),
			Consumed("suite", "AssertRetrySucceedsOnAttempt N-try recovery subtest"),
		),

		// Causality & ordering.
		New(SideEffect,
			Describe("causal relationship"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Method", ArgKey, Required),
			Consumed("suite", "AssertSideEffectObservable paired-method subtest"),
		),
		New(OrderAfter,
			Describe("call ordering constraint"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Method", ArgKey, Required),
			Consumed("stub", "AssertAfter check in dispatch (strict mode)"),
			Consumed("suite", "AssertOrderAfter precedence subtest"),
		),

		// Isolation.
		New(Partition,
			Describe("per-key isolation; in chain mode, the chain partitions by this field"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Field", ArgIdent, Required),
			Consumed("stub", "FaultForPartition / FaultForOtherPartitions helpers"),
			Consumed("suite", "AssertPartitionIsolation cross-partition fanout subtest"),
			Consumed("model", "in chain mode (replays), the field keys per-partition replay causality"),
		),

		// Input & validation.
		New(Validates,
			Describe("input validation"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Field", ArgIdent, Required),
			Consumed("suite", "AssertValidatesZeroInput zero-value rejection subtest"),
		),
		New(Bounded,
			Describe("return value bounds"),
			InCategory(Mixin),
			InPhase(Phase3),
			Arg("min..max", ArgRange, Required),
			Consumed("suite", "AssertBoundedRange in-range subtest"),
		),

		// Properties & invariants.
		New(Invariant,
			Describe("post-call property"),
			InCategory(Documentation),
			InPhase(Phase4),
			Arg("description", ArgString, Required, Multi),
		),

		// Testing & observability.
		New(Fuzz,
			Describe("generate fuzz target"),
			InCategory(Enrichment),
			InPhase(Phase4),
		),
		New(Hooks,
			Describe("fires callbacks"),
			InCategory(Enrichment),
			InPhase(Phase5),
			Arg("HookName", ArgIdent, Required, Multi),
			Consumed("suite", "AssertHooksFire registry-driven subtest"),
		),
		New(Req,
			Describe("requirement traceability"),
			InCategory(Documentation),
			InPhase(Phase1),
			Arg("REQ-ID", ArgString, Required, Multi),
		),

		// Consistency.
		New(Eventually,
			Describe("eventual consistency under merge; suite polls for convergence within the window"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("timeout", ArgDuration, Required),
			ConflictsWith(Linearizable, Causal, SnapshotIsolation),
			Consumed("suite", "AssertEventuallyConverges polling-based subtest (no time.Sleep)"),
			Consumed("model",
				"consistency-model selector; emits AUTO-EVENTUAL-CONVERGENCE; "+
					"mutually exclusive with linearizable/causal/snapshot-isolation"),
		),

		// Authorization.
		New(Scope,
			Describe("requires authorization"),
			InCategory(Enrichment),
			InPhase(Phase5),
			Arg("ScopeName", ArgIdent, Required),
			Consumed("suite", "AssertScopeAuthRequired unauthorized-rejection subtest"),
		),

		// Iteration.
		New(Pagination,
			Describe("paginated results with the named cursor field"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("CursorField", ArgIdent, Required),
			Consumed("suite", "AssertPaginates corpus-drain subtest"),
			Consumed("model", "promotes Reader to Paginator contract; emits AUTO-PAGINATOR-NO-DUPLICATES"),
		),

		// Shape hints.
		New(Deleter,
			Describe("marks method as delete-by-key shape"),
			InCategory(SignatureHint),
			InPhase(Phase1),
		),
		New(Mutator,
			Describe("explicit Mutator marker (auto-detected from signature)"),
			InCategory(SignatureHint),
			InPhase(Phase1),
		),
		New(NotMutator,
			Describe("opt-out of Mutator auto-detection"),
			InCategory(SignatureHint),
			InPhase(Phase1),
		),
		New(KeyField,
			Describe("key extraction field for reference synthesis"),
			InCategory(SignatureHint),
			InPhase(Phase1),
			Arg("FieldName", ArgIdent, Required),
		),

		// Smoke + bench input replacement.
		New(Sample,
			Describe("sample builder functions for non-context parameters"),
			InCategory(Enrichment),
			InPhase(Phase1),
			Arg("FuncName", ArgIdent, Required, Multi),
		),

		// Cross-method invariants — one descriptor per invariant; the
		// suite emits a paired-method contract subtest per directive.
		New(Cross,
			Describe("generic cross-method invariant declaration (escape hatch)"),
			InCategory(Enrichment),
			InPhase(Phase1),
			Arg("name", ArgIdent, Required),
			Arg("Methods", ArgKey, Required, Multi),
		),
		New(ReadAfterWrite,
			Describe("after this writer, the named reader returns the written value"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Reader", ArgIdent, Required),
			Consumed("suite", "AssertReadAfterWriteByKey paired write+read subtest"),
		),
		New(DeleteRemoves,
			Describe("after this deleter, the named reader returns the not-found sentinel"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Reader", ArgIdent, Required),
			Consumed("suite", "AssertDeleteRemovesByKey paired delete+read subtest"),
		),
		New(StreamReflectsMutations,
			Describe("after this writer, the named stream method yields the written value"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Stream", ArgIdent, Required),
			Consumed("suite", "AssertStreamReflectsValueWritten paired write+drain subtest"),
		),
		New(LifecycleAfterClose,
			Describe("after this close, the named reader returns the closed sentinel"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Reader", ArgIdent, Required),
			Consumed("suite", "AssertLifecycleAfterCloseReflective paired close+read subtest"),
		),
		New(CRDTMerge,
			Describe("two impls applying operations in opposite orders converge to equal state"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Other", ArgIdent, Required),
			Consumed("suite", "AssertCRDTMerge dual-impl convergence subtest"),
		),

		// Sentinel cross-package non-overlap.
		New(SentinelNoOverlapWith,
			Describe("declare additional packages to verify sentinel non-overlap with"),
			InCategory(Enrichment),
			InPhase(Phase1),
			Arg("ImportPath", ArgString, Required, Multi),
		),

		// Per-field default literal (builder).
		New(Default,
			Describe("seed value for a builder field when no Defaults factory exists"),
			InCategory(Enrichment),
			InPhase(Phase1),
			Arg("Value", ArgString, Required),
		),

		// Chain mode. Carrier method appends to a chain; consumers
		// emit hash-chain integrity, replay determinism, and
		// causality-respecting laws.
		New(NonDeterministic,
			Describe("output is non-deterministic; suppress determinism laws"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Consumed("model", "suppresses AUTO-PURE-DETERMINISTIC / AUTO-PREDICATE-CONSISTENT"),
		),
		New(TimeAware,
			Describe("interface depends on the test clock; emit clock-factory option"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Consumed("model", "emits StoreModelClockFactory + advance actions + AUTO-TTL-EXPIRY law"),
		),
		New(Appends,
			Describe("writer-class carrier of an append-only chain"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Consumed("model", "emits AUTO-APPEND-ONLY-GROWS + AUTO-HASH-CHAIN-INTEGRITY"),
		),
		New(Replays,
			Describe("StreamReader carrier replays the appended chain"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Consumed("model", "emits AUTO-APPEND-ONLY-NO-DROPS + AUTO-REPLAY-DETERMINISTIC"),
		),
		New(Verifies,
			Describe("Lifecycle/PoisonAccessor that checks chain integrity"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Consumed("model", "binds AUTO-HASH-CHAIN-INTEGRITY-VIA-VERIFY"),
		),
		New(EntryID,
			Describe("entry identifier for replay causality"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("Field", ArgIdent, Required),
			Requires(Replays, DependsOn),
		),
		New(DependsOn,
			Describe("predecessor reference for replay causality"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("Field", ArgIdent, Required),
			Requires(Replays, EntryID),
		),
		New(Hash,
			Describe("explicit hash function for the append-only chain"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("PkgFunc", ArgString, Required),
			ComposesWith(Appends),
		),

		// Contract-tier promotions. Each names a sibling method or
		// field whose existence the codegen-time validator resolves;
		// mismatch is a hard error.
		New(Persister,
			Describe("Writer-with-result promoted to Persister: retrievable by returned ID"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Arg("Reader", ArgIdent, Required),
			Consumed("model", "emits AUTO-PERSISTER-RETRIEVABLE"),
		),
		New(Updater,
			Describe("Writer/CompositeWriter promoted to Updater: replaces by key"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Arg("Reader", ArgIdent, Required),
			Consumed("model", "emits AUTO-UPDATER-REPLACES"),
		),
		New(Upserter,
			Describe("idempotent insert-or-update"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Arg("Reader", ArgIdent, Required),
			Consumed("model", "emits AUTO-UPSERTER-IDEMPOTENT"),
		),
		New(CAS,
			Describe("compare-and-swap with version field; exactly-one-winner under concurrency"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Arg("VersionField", ArgIdent, Required),
			Consumed("model", "emits AUTO-CAS-ATOMIC-ONE-WINNER + selects linearize.CASCell"),
		),
		New(Appender,
			Describe("Writer that returns monotonic offsets; gap-free append"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Consumed("model", "emits AUTO-APPENDER-MONOTONIC-OFFSETS"),
		),
		New(Watcher,
			Describe("subscribes to changes triggered by the named method"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Arg("Trigger", ArgIdent, Required),
			Consumed("model", "emits AUTO-WATCHER-RETURNS-ON-CHANGE"),
		),
		New(Singleflight,
			Describe("N concurrent same-key calls invoke the supplied func once"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Consumed("model", "emits AUTO-SINGLEFLIGHT-COALESCES"),
		),
		New(Transaction,
			Describe("transactional func: rollback on error, no mid-tx visibility"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Consumed("model", "emits AUTO-TRANSACTION-ROLLBACK + AUTO-TRANSACTION-NO-MID-TX-VISIBILITY"),
		),
		New(Acquire,
			Describe("lease/lock acquire paired with the named release method"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Arg("Release", ArgIdent, Required),
			Consumed("model", "emits AUTO-LEASE-RELEASED-ON-CANCEL + AUTO-LEASE-DOUBLE-ACQUIRE-BLOCKS"),
		),
		New(Publisher,
			Describe("publishes to subscribers reached via the named subscribe method"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Arg("Subscribe", ArgIdent, Required),
			Consumed("model", "emits AUTO-PUBLISHER-DELIVERS"),
		),
		New(Subscribe,
			Describe("channel- or callback-based subscriber paired with a Publisher"),
			InCategory(ContractTier),
			InPhase(Phase4),
		),

		// Composite-tier promotions. Multi-method shapes where the
		// directive names the paired method(s).
		New(Pool,
			Describe("Get/Put pair: leak-free, balanced, double-put rejected"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Arg("Put", ArgIdent, Required),
			Consumed("model", "emits AUTO-POOL-LEAK-FREE + AUTO-POOL-BALANCED"),
		),
		New(Cursor,
			Describe("Next/Close pair: close idempotent, next-after-close → sentinel"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Arg("Close", ArgIdent, Required),
			Consumed("model", "emits AUTO-CURSOR-CLOSE-IDEMPOTENT + AUTO-CURSOR-NEXT-AFTER-CLOSE"),
		),
		New(TwoPhase,
			Describe("Begin returning a Tx with paired Commit/Rollback methods"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Arg("Commit", ArgIdent, Required),
			Arg("Rollback", ArgIdent, Required),
			Consumed("model", "emits AUTO-TWO-PHASE-MUTEX + AUTO-TWO-PHASE-ROLLBACK-AFTER-COMMIT"),
		),
		New(Saga,
			Describe("multi-step chain with full compensation on partial failure"),
			InCategory(ContractTier),
			InPhase(Phase4),
			Arg("Steps", ArgIdent, Required, Multi),
			Consumed("model", "emits AUTO-SAGA-FULL-COMPENSATION"),
		),

		// Mixin axis. Additional mixins; the existing atomic/idempotent/
		// monotonic/pure/nilsafe/cacheable/concurrent/bounded mixins
		// are registered above.
		New(Commutative,
			Describe("operation order does not affect final state"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(Associative,
			Describe("operation grouping does not affect final state"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(Permutation,
			Describe("output is a permutation of input"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(Windowed,
			Describe("rolling-window state across the named duration"),
			InCategory(Mixin),
			InPhase(Phase4),
			Arg("Duration", ArgDuration, Required),
		),
		New(CASAtomic,
			Describe("Writer with version field; promotes to CompareAndSwap"),
			InCategory(Mixin),
			InPhase(Phase4),
			Arg("VersionField", ArgIdent, Required),
		),
		New(PointInTime,
			Describe("Reader returns snapshot semantics; reads see the moment, not now"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(LeakFree,
			Describe("no goroutine or FD leaks across cycles"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(TamperEvident,
			Describe("post-write detection of modification"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(ConstantTime,
			Describe("no timing side channels"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(Delivery,
			Describe("Publisher delivery guarantee selector"),
			InCategory(Mixin),
			InPhase(Phase4),
			Arg("Mode", ArgEnum, Required, OneOf("at-least-once", "at-most-once", "exactly-once")),
		),
		New(DefaultOnError,
			Describe("Reader returns the supplied default expression on error; never panics"),
			InCategory(Mixin),
			InPhase(Phase4),
			Arg("DefaultExpr", ArgString, Required),
		),
		New(ValidTransitionOnly,
			Describe("Mutator/Writer respects a state-machine constraint on the named field"),
			InCategory(Mixin),
			InPhase(Phase4),
			Arg("Field", ArgIdent, Required),
		),
		New(OverMatchAcceptable,
			Describe("stream may yield extra elements; missing elements fail"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(XSSSafe,
			Describe("input is sanitized for HTML output"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(InjectionSafe,
			Describe("input is parameterized for SQL/shell"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(TotalOver,
			Describe("Pure/Aggregator defined for every input in the named domain"),
			InCategory(Mixin),
			InPhase(Phase4),
			Arg("Domain", ArgIdent, Required),
		),
		New(PaginationResumable,
			Describe("Paginator resumable from any cursor"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(Roundtrip,
			Describe("Inverse(F(x)) == x"),
			InCategory(Mixin),
			InPhase(Phase4),
			Arg("Inverse", ArgIdent, Required),
		),
		New(LossyRoundtrip,
			Describe("F(Inverse(F(x))) == F(x)"),
			InCategory(Mixin),
			InPhase(Phase4),
			Arg("Inverse", ArgIdent, Required),
		),
		New(StableOrder,
			Describe("output order preserves input order"),
			InCategory(Mixin),
			InPhase(Phase4),
		),
		New(Sticky,
			Describe("first-call result caches on the named key"),
			InCategory(Mixin),
			InPhase(Phase4),
			Arg("Key", ArgIdent, Required),
		),
		New(Conservative,
			Describe("Mutator+Aggregator pair preserves a sum-of-Field invariant"),
			InCategory(Mixin),
			InPhase(Phase4),
			Arg("Field", ArgIdent, Required),
		),

		// Consistency-model selectors. linearizable, eventually,
		// causal, and snapshot-isolation are mutually exclusive at the
		// interface level; per-client guarantees (read-your-writes,
		// monotonic-reads, monotonic-writes, writes-follow-reads)
		// compose with any of the others.
		New(Linearizable,
			Describe("explicit linearizability assertion (default for mutation shapes)"),
			InCategory(Enrichment),
			InPhase(Phase4),
			ConflictsWith(Eventually, Causal, SnapshotIsolation),
		),
		New(Causal,
			Describe("non-linearizable orderings allowed if causally consistent"),
			InCategory(Enrichment),
			InPhase(Phase4),
			ConflictsWith(Linearizable, Eventually, SnapshotIsolation),
		),
		New(SnapshotIsolation,
			Describe("no mid-tx visibility; rejects G0/G1/G2 anomalies"),
			InCategory(Enrichment),
			InPhase(Phase4),
			ConflictsWith(Linearizable, Eventually, Causal),
		),
		New(ReadYourWrites,
			Describe("per-client read-your-writes guarantee"),
			InCategory(Enrichment),
			InPhase(Phase4),
		),
		New(MonotonicReads,
			Describe("per-client monotonic-reads guarantee"),
			InCategory(Enrichment),
			InPhase(Phase4),
		),
		New(MonotonicWrites,
			Describe("per-client monotonic-writes guarantee"),
			InCategory(Enrichment),
			InPhase(Phase4),
		),
		New(WritesFollowReads,
			Describe("per-client writes-follow-reads guarantee"),
			InCategory(Enrichment),
			InPhase(Phase4),
		),

		// Action distribution & precondition.
		New(Bias,
			Describe("property-engine command-distribution weight for this method"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("Weight", ArgString, Required),
		),
		New(Precondition,
			Describe("Go boolean expression that must hold for the action to fire"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("Expr", ArgString, Required),
		),
		New(Expensive,
			Describe("law fires on a sampled subset of iterations"),
			InCategory(Enrichment),
			InPhase(Phase4),
		),

		// Fault-injection control.
		New(FaultWindowOf,
			Describe("inject faults during execution of the named method, not before/after"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("Method", ArgKey, Required),
		),

		// Model-internal.
		New(Reference,
			Describe("override reference primitive selection"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("PkgPathName", ArgString, Required),
		),
		New(DomainGen,
			Describe("register a domain-typed generator"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("Type", ArgIdent, Required),
			Arg("Func", ArgIdent, Required),
		),
		New(Clock,
			Describe("declare an additional named TestClock (e.g., wall, logical)"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("Name", ArgIdent, Required),
		),
	}
}

// defaultRegistry is initialized lazily on first access. Tests that
// need isolation construct a fresh Registry via [NewRegistry] and
// register their own descriptors.
var (
	defaultOnce     sync.Once
	defaultRegistry *Registry
)

// DefaultRegistry returns the package-level [Registry] populated with
// every known directive descriptor. The registry is built once on
// first access; subsequent calls return the same instance.
//
// After all descriptors are registered the registry computes the
// transitive closure of [Descriptor.Implies] — a descriptor that
// implies another inherits the implied descriptor's Conflicts and
// Requires. This means [Cacheable] does not need to repeat
// [Pure]'s conflicts; declaring `Implies(Pure)` is sufficient.
func DefaultRegistry() *Registry {
	defaultOnce.Do(func() {
		defaultRegistry = NewRegistry()
		for _, d := range defaultDescriptors() {
			defaultRegistry.MustRegister(d)
		}
		defaultRegistry.closeImplications()
	})
	return defaultRegistry
}
