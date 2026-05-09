// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coveragetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/coverage"
	"go.thesmos.sh/testkit/suite"
)

// TestCoverageContract closes the e2e loop on every directive that
// the AllShapes / Directives fixtures don't naturally exercise.
// The factory returns a fresh in-mem with state pre-aligned to the
// shape-baseline samples (Description="test-result0",
// version=42, etc.); the option suite wires the directive-specific
// runtime helpers (HookRecorder, ScopeContext, AggregatorBounds,
// LeaseRelease).
func TestCoverageContract(t *testing.T) {
	t.Parallel()
	AssertCoverageContract(
		t,
		func() coverage.Coverage { return coverage.NewInMem() },
		// Bounded: Version's [0..1000] range. The companion supplies
		// the bound; Version returns 42 ∈ [0..1000] so the assertion
		// passes.
		suite.WithAggregatorBounds(0, 1000),
		// Scope: Privileged returns ErrUnauthorized when ctx lacks
		// the scope key. WithScopeContext provides the typed
		// decorator; WithScopeUnauthorized declares the sentinel.
		suite.WithScopeContext(coverage.ScopeContext(t.Context())),
		suite.WithScopeUnauthorized(coverage.ErrUnauthorized),
		// Lease: AcquireLease + ReleaseLease pair. The runtime
		// helper resolves both ends via reflection on the configured
		// release-method name.
		suite.WithLeaseRelease("ReleaseLease"),
	)
}
