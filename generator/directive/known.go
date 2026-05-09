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
	Allocs  = "allocs"
	Latency = "latency"

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

	// Cross-method invariants.
	Cross = "cross"

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
		New(Errors,
			Describe("sentinel error returns"),
			InCategory(Enrichment),
			InPhase(Phase1),
			Arg("ErrName", ArgIdent, Required, Multi),
			Consumed("stub", "Fault<Sentinel>() helper per name"),
		),
		New(WrappedVia,
			Describe("error wrapping discipline"),
			InCategory(Enrichment),
			InPhase(Phase1),
			Arg("ErrName", ArgIdent, Required),
			Requires(Errors),
			Consumed("stub", "Fault<Sentinel> helpers wrap via target"),
		),

		// Behavioral properties (Mixin tier).
		New(Idempotent,
			Describe("repeated calls produce same result"),
			InCategory(Mixin),
			InPhase(Phase1),
		),
		New(Pure,
			Describe("no side effects"),
			InCategory(Mixin),
			InPhase(Phase1),
			ConflictsWith(SideEffect, Monotonic),
			Implies(Idempotent),
		),
		New(Cacheable,
			Describe("deterministic function of inputs"),
			InCategory(Mixin),
			InPhase(Phase3),
			Implies(Pure),
		),
		New(Monotonic,
			Describe("ordered results"),
			InCategory(Mixin),
			InPhase(Phase3),
		),

		// Safety (Mixin tier).
		New(Concurrent,
			Describe("safe for parallel access"),
			InCategory(Mixin),
			InPhase(Phase1),
			ConflictsWith(ConcurrentReaders),
		),
		New(ConcurrentReaders,
			Describe("parallel reads, serialised writes"),
			InCategory(Mixin),
			InPhase(Phase3),
		),
		New(NilSafe,
			Describe("zero/nil inputs do not panic"),
			InCategory(Mixin),
			InPhase(Phase1),
		),
		New(Atomic,
			Describe("all-or-nothing"),
			InCategory(Mixin),
			InPhase(Phase3),
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
		),
		New(Deprecated,
			Describe("method is deprecated"),
			InCategory(Enrichment),
			InPhase(Phase5),
			Arg("Replacement", ArgString, Required),
			Consumed("stub", "tb.Logf in dispatch + // Deprecated: doc comment"),
		),
		New(Lease,
			Describe("acquires resource, must release"),
			InCategory(Enrichment),
			InPhase(Phase4),
		),
		New(IntegrationOnly,
			Describe("opt out of stubbing"),
			InCategory(Enrichment),
			InPhase(Phase5),
			Consumed("stub", "skip dispatch (zero return, no record)"),
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

		// Resilience.
		New(Retryable,
			Describe("safe to retry"),
			InCategory(Mixin),
			InPhase(Phase3),
		),
		New(RetrySucceedsOnAttempt,
			Describe("transient-failure recovery"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("N", ArgInt, Required),
			Requires(Retryable),
			Consumed("stub", "RetrySchedule(err) helper"),
		),

		// Causality & ordering.
		New(SideEffect,
			Describe("causal relationship"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Method", ArgKey, Required),
		),
		New(OrderAfter,
			Describe("call ordering constraint"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Method", ArgKey, Required),
			Consumed("stub", "AssertAfter check in dispatch (strict mode)"),
		),

		// Isolation.
		New(Partition,
			Describe("per-key isolation"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Field", ArgIdent, Required),
			Consumed("stub", "FaultForPartition / FaultForOtherPartitions helpers"),
		),

		// Input & validation.
		New(Validates,
			Describe("input validation"),
			InCategory(Enrichment),
			InPhase(Phase3),
			Arg("Field", ArgIdent, Required),
		),
		New(Bounded,
			Describe("return value bounds"),
			InCategory(Mixin),
			InPhase(Phase3),
			Arg("min..max", ArgRange, Required),
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
		),

		// Authorization.
		New(Scope,
			Describe("requires authorization"),
			InCategory(Enrichment),
			InPhase(Phase5),
			Arg("ScopeName", ArgIdent, Required),
		),

		// Iteration.
		New(Pagination,
			Describe("paginated results"),
			InCategory(Enrichment),
			InPhase(Phase4),
			Arg("CursorField", ArgIdent, Required),
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

		// Cross-method invariants.
		New(Cross,
			Describe("cross-method invariant"),
			InCategory(Enrichment),
			InPhase(Phase1),
			Arg("name", ArgIdent, Required),
			Arg("Methods", ArgKey, Required, Multi),
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
