// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ledgertest_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/model/testdata/thesmos"
	"go.thesmos.sh/testkit/gen/model/testdata/thesmos/ledgertest"
)

func TestLedgerModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 0 per-RunID partitioned chain", func(t *testing.T) {
		t.Parallel()
		// Tier 0: refchain.PartitionedAppendOnly auto-synthesized.
		// Per-RunID chain laws fire: appends to run-1 isolated from run-2.
		ledgertest.AssertLedgerModel(t,
			func() thesmos.Ledger { return thesmos.NewInMemoryLedger() },
		)
	})

	t.Run("catches dropped entries", func(t *testing.T) {
		t.Parallel()
		// Negative: BrokenLedger silently drops every 3rd entry.
		// Chain laws catch the gap between attempted appends and replay.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			ledgertest.AssertLedgerModel(ft,
				func() thesmos.Ledger { return thesmos.NewBrokenLedger() },
			)
		}()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("negative ledger test timed out")
		}
		if !ft.Failed() {
			t.Fatal("framework should have caught dropped entries for RunID")
		}
	})
}
