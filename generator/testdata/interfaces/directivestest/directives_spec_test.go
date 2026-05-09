// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directivestest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/interfaces"
	"go.thesmos.sh/testkit/suite"
)

// TestDirectivesContract closes the e2e loop on the suite generator's
// directive coverage. The fixture exercises the directives whose
// contracts are most likely to interact with shape baselines:
// errors, wrapped-via, deprecated, retryable + retry-succeeds-on-
// attempt, order-after, partition.
//
// The seed function pre-populates the items map under the contract's
// sample key so Read's Reader baseline lands on the sample value;
// other methods (Submit / Wrap / Retry / Shard / ShardByKey) honor
// their declared sentinels via the in-mem's branch on zero-valued
// inputs.
//
// Two factories are wired into the contract:
//   - The default factory returns a fresh in-mem with retryFailMode=
//     false so the Writer baseline (Submit / Wrap / Retry / Shard /
//     ShardByKey) sees first-call success.
//   - WithRetryFactory supplies a separate in-mem with retryFailMode=
//     true so AssertRetrySucceedsOnAttempt observes the N-1
//     transient failures the directive declares.
func TestDirectivesContract(t *testing.T) {
	t.Parallel()
	AssertDirectivesContract(t, func() interfaces.Directives {
		d := interfaces.NewInMemoryDirectives()
		// Read's Reader baseline expects items["test-key"] =
		// Record{ID:"test-id"} so the happy-path read returns the
		// sample value.
		d.SeedAt("test-key", interfaces.Record{ID: "test-id"})
		return d
	}, suite.WithRetryFactory(func() interfaces.Directives {
		// Retry contract requires N-1 transient failures + final
		// success against the same impl. The retry factory builds an
		// in-mem with retryFailMode=true so Retry returns ErrInternal
		// on the first 2 calls and succeeds on the 3rd.
		d := interfaces.NewInMemoryDirectives()
		d.SetRetryFailMode(true)
		return d
	}))
}
