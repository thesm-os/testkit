// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

// TestKnownDescriptors guards the canonical descriptor table against
// regressions: every shipped name resolves, every category has at
// least one entry, and the cacheable→pure→idempotent implication
// chain produces the expected closure.
func TestKnownDescriptors(t *testing.T) {
	t.Parallel()

	r := directive.DefaultRegistry()

	t.Run("every name constant resolves to a descriptor", func(t *testing.T) {
		t.Parallel()
		names := []string{
			directive.Errors, directive.WrappedVia,
			directive.Idempotent, directive.Pure, directive.Cacheable, directive.Monotonic,
			directive.Concurrent, directive.ConcurrentReaders, directive.NilSafe, directive.Atomic,
			directive.Ctx, directive.Timeout, directive.Deprecated, directive.Lease, directive.IntegrationOnly,
			directive.Allocs, directive.Latency, directive.Percentiles,
			directive.Retryable, directive.RetrySucceedsOnAttempt,
			directive.SideEffect, directive.OrderAfter,
			directive.Partition,
			directive.Validates, directive.Bounded,
			directive.Invariant,
			directive.Fuzz, directive.Hooks, directive.Req,
			directive.Eventually,
			directive.Scope,
			directive.Pagination,
			directive.Deleter, directive.Mutator, directive.NotMutator, directive.KeyField,
			directive.Sample,
			directive.Cross,
		}
		for _, n := range names {
			testkit.True(t, r.IsKnown(n), "name registered: "+n)
		}
	})

	t.Run("every model-owned name constant resolves to a descriptor", func(t *testing.T) {
		t.Parallel()
		names := []string{
			// Chain mode.
			directive.NonDeterministic, directive.TimeAware,
			directive.Appends, directive.Replays, directive.Verifies,
			directive.EntryID, directive.DependsOn, directive.Hash,

			// Contract-tier promotions.
			directive.Persister, directive.Updater, directive.Upserter,
			directive.CAS, directive.Appender, directive.Watcher,
			directive.Singleflight, directive.Transaction, directive.Acquire,
			directive.Publisher, directive.Subscribe,

			// Composite-tier promotions.
			directive.Pool, directive.Cursor, directive.TwoPhase, directive.Saga,

			// Mixin axis.
			directive.Commutative, directive.Associative, directive.Permutation,
			directive.Windowed, directive.CASAtomic, directive.PointInTime,
			directive.LeakFree, directive.TamperEvident, directive.ConstantTime,
			directive.Delivery, directive.DefaultOnError, directive.ValidTransitionOnly,
			directive.OverMatchAcceptable, directive.XSSSafe, directive.InjectionSafe,
			directive.TotalOver, directive.PaginationResumable,
			directive.Roundtrip, directive.LossyRoundtrip, directive.StableOrder,
			directive.Sticky, directive.Conservative,

			// Consistency-model selectors.
			directive.Linearizable, directive.Causal, directive.SnapshotIsolation,
			directive.ReadYourWrites, directive.MonotonicReads,
			directive.MonotonicWrites, directive.WritesFollowReads,

			// Action distribution & precondition.
			directive.Bias, directive.Precondition, directive.Expensive,

			// Fault-injection control.
			directive.FaultWindowOf,

			// Model-internal.
			directive.Reference, directive.DomainGen, directive.Clock,
		}
		for _, n := range names {
			testkit.True(t, r.IsKnown(n), "name registered: "+n)
		}
	})

	t.Run("chain-mode entry-id and depends-on require each other plus replays", func(t *testing.T) {
		t.Parallel()
		eid, ok := r.Get(directive.EntryID)
		testkit.True(t, ok, "entry-id descriptor")
		testkit.Assert(t, eid.Requires).
			Contains(directive.Replays, "entry-id requires replays").
			Contains(directive.DependsOn, "entry-id requires depends-on")

		dep, ok := r.Get(directive.DependsOn)
		testkit.True(t, ok, "depends-on descriptor")
		testkit.Assert(t, dep.Requires).
			Contains(directive.Replays, "depends-on requires replays").
			Contains(directive.EntryID, "depends-on requires entry-id")
	})

	t.Run("consistency-model selectors are mutually exclusive", func(t *testing.T) {
		t.Parallel()
		selectors := []string{
			directive.Linearizable,
			directive.Eventually,
			directive.Causal,
			directive.SnapshotIsolation,
		}
		for _, name := range selectors {
			desc, ok := r.Get(name)
			testkit.True(t, ok, name+" descriptor")
			for _, other := range selectors {
				if other == name {
					continue
				}
				testkit.Assert(t, desc.Conflicts).Contains(other,
					name+" conflicts with "+other)
			}
		}
	})

	t.Run("delivery enforces a fixed mode set", func(t *testing.T) {
		t.Parallel()
		desc, ok := r.Get(directive.Delivery)
		testkit.True(t, ok, "delivery descriptor")
		testkit.Equal(t, len(desc.Args), 1, "one arg slot")
		testkit.Equal(t, desc.Args[0].Kind, directive.ArgEnum, "enum kind")
		testkit.Equal(t, desc.Args[0].Enum,
			[]string{"at-least-once", "at-most-once", "exactly-once"},
			"three delivery modes")
	})

	t.Run("transitive Implies closure pulls Pure's conflicts onto Cacheable", func(t *testing.T) {
		t.Parallel()
		desc, ok := r.Get("cacheable")
		testkit.True(t, ok, "cacheable descriptor")
		hasMonotonic := false
		hasSideEffect := false
		for _, c := range desc.Conflicts {
			if c == "monotonic" {
				hasMonotonic = true
			}
			if c == "sideeffect" {
				hasSideEffect = true
			}
		}
		testkit.True(t, hasMonotonic, "cacheable → pure → monotonic")
		testkit.True(t, hasSideEffect, "cacheable → pure → sideeffect")
	})

	t.Run("retry-succeeds-on-attempt requires retryable", func(t *testing.T) {
		t.Parallel()
		desc, ok := r.Get("retry-succeeds-on-attempt")
		testkit.True(t, ok, "descriptor exists")
		hasReq := false
		for _, n := range desc.Requires {
			if n == "retryable" {
				hasReq = true
			}
		}
		testkit.True(t, hasReq, "retryable in Requires")
	})
}
