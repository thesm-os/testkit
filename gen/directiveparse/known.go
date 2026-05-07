// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directiveparse

const argFieldName = "FieldName"

// Generator name constants for directive descriptors.
const (
	genStub  = "stub"
	genSuite = "suite"
	genModel = "model"
	genBench = "bench"
)

// DefaultRegistry returns a Registry pre-populated with all known
// testkit directives. Used by generators for unknown-directive
// validation (strict-by-default).
func DefaultRegistry() *Registry {
	r := NewRegistry()
	for _, d := range knownDirectives {
		r.Register(d)
	}
	return r
}

// knownDirectives lists all 31 directives from the spec.
//
//nolint:funlen // declarative table, not complex logic
var knownDirectives = []Descriptor{
	// Error & return contract
	{
		Name: "errors", Description: "sentinel error returns",
		Args:       "ErrName [ErrName...]",
		Generators: []string{genStub, genSuite, genModel}, Phase: 1,
	},
	{
		Name: "wrapped-via", Description: "error wrapping discipline",
		Args:       "ErrName",
		Generators: []string{genSuite, genModel}, Phase: 1,
	},

	// Behavioral properties
	{
		Name: "idempotent", Description: "repeated calls produce same result",
		Generators: []string{genSuite, genModel}, Phase: 1,
	},
	{
		Name: "pure", Description: "no side effects",
		Generators: []string{genSuite, genModel}, Phase: 1,
	},
	{
		Name: "cacheable", Description: "deterministic function of inputs",
		Generators: []string{genSuite, genModel}, Phase: 3,
	},
	{
		Name: "monotonic", Description: "ordered results",
		Generators: []string{genSuite, genModel}, Phase: 3,
	},

	// Safety
	{
		Name: "concurrent", Description: "safe for parallel access",
		Generators: []string{genSuite, genModel}, Phase: 1,
	},
	{
		Name: "concurrent-readers", Description: "parallel reads, serialised writes",
		Generators: []string{genSuite, genModel}, Phase: 3,
	},
	{
		Name: "nilsafe", Description: "zero/nil inputs do not panic",
		Generators: []string{genSuite}, Phase: 1,
	},
	{
		Name: "atomic", Description: "all-or-nothing",
		Generators: []string{genSuite, genModel}, Phase: 3,
	},

	// Context & lifecycle
	{
		Name: "ctx", Description: "respects context cancellation",
		Generators: []string{genSuite}, Phase: 1,
	},
	{
		Name: "timeout", Description: "must complete within deadline",
		Generators: []string{genSuite}, Phase: 1,
	},
	{
		Name: "deprecated", Description: "method is deprecated",
		Args:       "Replacement",
		Generators: []string{genStub, genSuite}, Phase: 5,
	},
	{
		Name: "lease", Description: "acquires resource, must release",
		Generators: []string{genSuite}, Phase: 4,
	},
	{
		Name: "integration-only", Description: "opt out of stubbing",
		Generators: []string{genStub, genSuite, genModel}, Phase: 5,
	},

	// Performance
	{
		Name: "allocs", Description: "allocation ceiling",
		Args: "N", Generators: []string{genBench}, Phase: 2,
	},
	{
		Name: "latency", Description: "latency ceiling",
		Args: "Xms", Generators: []string{genBench}, Phase: 2,
	},

	// Resilience
	{
		Name: "retryable", Description: "safe to retry",
		Generators: []string{genSuite, genModel}, Phase: 3,
	},
	{
		Name: "retry-succeeds-on-attempt", Description: "transient-failure recovery",
		Args: "N", Generators: []string{genStub, genSuite}, Phase: 3,
	},

	// Causality & ordering
	{
		Name: "sideeffect", Description: "causal relationship",
		Args: "Method", Generators: []string{genSuite, genModel}, Phase: 3,
	},
	{
		Name: "order-after", Description: "call ordering constraint",
		Args: "Method", Generators: []string{genStub, genSuite}, Phase: 3,
	},

	// Isolation
	{
		Name: "partition", Description: "per-key isolation",
		Args: "Field", Generators: []string{genSuite, genModel}, Phase: 3,
	},

	// Input & validation
	{
		Name: "validates", Description: "input validation",
		Args: "Field", Generators: []string{genSuite}, Phase: 3,
	},
	{
		Name: "bounded", Description: "return value bounds",
		Args: "min max", Generators: []string{genSuite, genModel}, Phase: 3,
	},

	// Properties & invariants
	{
		Name: "invariant", Description: "post-call property",
		Args: "description", Generators: []string{genSuite, genModel}, Phase: 4,
	},

	// Testing & observability
	{
		Name: "fuzz", Description: "generate fuzz target",
		Generators: []string{genSuite}, Phase: 4,
	},
	{
		Name: "hooks", Description: "fires callbacks",
		Args: "HookName", Generators: []string{genStub, genSuite}, Phase: 5,
	},
	{
		Name: "req", Description: "requirement traceability",
		Args: "REQ-ID", Generators: []string{genSuite, genModel}, Phase: 1,
	},

	// Consistency
	{
		Name: "eventually", Description: "eventual consistency",
		Args: "timeout", Generators: []string{genSuite, genModel}, Phase: 4,
	},

	// Authorization
	{
		Name: "scope", Description: "requires authorization",
		Args: "ScopeName", Generators: []string{genStub, genSuite}, Phase: 5,
	},

	// Iteration
	{
		Name: "pagination", Description: "paginated results",
		Args: "CursorField", Generators: []string{genSuite, genModel}, Phase: 4,
	},

	// Shape hints
	{
		Name: "deleter", Description: "marks method as delete-by-key shape",
		Generators: []string{genStub, genSuite, genModel}, Phase: 1,
	},
	{
		Name: "mutator", Description: "marks method as state-mutating command with no return",
		Generators: []string{genModel}, Phase: 1,
	},
	{
		Name: "keyfield", Description: "key extraction field for reference synthesis",
		Args: argFieldName, Generators: []string{genModel}, Phase: 1,
	},

	// Chain directives (Pillar 4)
	{
		Name: "appends", Description: "marks method as chain append operation",
		Generators: []string{genModel}, Phase: 1,
	},
	{
		Name: "verifies", Description: "marks method as chain integrity verifier",
		Generators: []string{genModel}, Phase: 1,
	},
	{
		Name: "replays", Description: "marks method as chain replay/stream operation",
		Generators: []string{genModel}, Phase: 1,
	},
	{
		Name: "partition-by", Description: "field on entry struct used as partition key",
		Args: argFieldName, Generators: []string{genModel}, Phase: 1,
	},
	{
		Name: "entry-id", Description: "field on entry struct used as unique entry ID for causal ordering",
		Args: argFieldName, Generators: []string{genModel}, Phase: 1,
	},
	{
		Name: "depends-on", Description: "field on entry struct listing dependency IDs for causal ordering",
		Args: argFieldName, Generators: []string{genModel}, Phase: 1,
	},
	{
		Name: "hash", Description: "qualified function name for custom chain hash",
		Args: "PkgPath.FuncName", Generators: []string{genModel}, Phase: 1,
	},
	{
		Name: "time-aware", Description: "inject TestClock pair for deterministic time testing",
		Generators: []string{genModel}, Phase: 1,
	},
	{
		Name: "nondeterministic", Description: "suppress determinism auto-laws for this method",
		Generators: []string{genModel}, Phase: 1,
	},
	{
		Name: "sample", Description: "sample builder functions for non-context parameters",
		Args: "FuncName [FuncName...]", Generators: []string{genSuite, genBench}, Phase: 1,
	},
}
