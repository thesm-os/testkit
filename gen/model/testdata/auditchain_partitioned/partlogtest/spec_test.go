// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package partlogtest_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	ap "go.thesmos.sh/testkit/gen/model/testdata/auditchain_partitioned"
	"go.thesmos.sh/testkit/gen/model/testdata/auditchain_partitioned/partlogtest"
)

func TestPartitionedLogModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 0 partitioned chain", func(t *testing.T) {
		t.Parallel()
		// Tier 0: refchain.PartitionedAppendOnly auto-synthesized.
		// Chain laws fire per-partition: appends to tenant A don't
		// affect tenant B's replay.
		partlogtest.AssertPartitionedLogModel(t,
			func() ap.PartitionedLog { return ap.NewInMemoryPartitionedLog() },
		)
	})

	t.Run("catches cross-partition leak", func(t *testing.T) {
		t.Parallel()
		// Negative: LeakyPartitionedLog routes all entries to "leaked"
		// partition. Replay("tenantA") returns empty while history
		// recorded appends for "tenantA" → AppendOnlyNoDrops fires.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			partlogtest.AssertPartitionedLogModel(ft,
				func() ap.PartitionedLog { return ap.NewLeakyPartitionedLog() },
			)
		}()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("negative partition test timed out")
		}
		if !ft.Failed() {
			t.Fatal("framework should have caught cross-partition leak")
		}
	})
}
