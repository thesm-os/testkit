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
			Describe("per-key isolation"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Field", ArgIdent, Required),
			Consumed("stub", "FaultForPartition / FaultForOtherPartitions helpers"),
			Consumed("suite", "AssertPartitionIsolation cross-partition fanout subtest"),
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
			Describe("eventual consistency"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("timeout", ArgDuration, Required),
			Consumed("suite", "AssertEventuallyConverges polling-based subtest (no time.Sleep)"),
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
			Describe("paginated results"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("CursorField", ArgIdent, Required),
			Consumed("suite", "AssertPaginates corpus-drain subtest"),
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
