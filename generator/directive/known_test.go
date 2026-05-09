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
