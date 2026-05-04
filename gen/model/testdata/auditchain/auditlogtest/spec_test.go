// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package auditlogtest_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/model/testdata/auditchain"
	"go.thesmos.sh/testkit/gen/model/testdata/auditchain/auditlogtest"
)

func TestAuditLogModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 0 chain laws fire end to end", func(t *testing.T) {
		t.Parallel()
		// Tier 0: no consumer-supplied reference. refchain.AppendOnly
		// auto-synthesized. Chain laws AUTO-APPEND-ONLY-GROWS,
		// AUTO-HASH-CHAIN-INTEGRITY, AUTO-APPEND-ONLY-NO-DROPS,
		// AUTO-REPLAY-DETERMINISTIC all fire.
		auditlogtest.AssertAuditLogModel(t,
			func() auditchain.AuditLog { return auditchain.NewInMemoryAuditLog() },
		)
	})

	t.Run("catches broken append that drops entries", func(t *testing.T) {
		t.Parallel()
		// Negative: BrokenAuditLog drops every 3rd entry.
		// AUTO-APPEND-ONLY-GROWS detects the gap: SUT chain grows
		// but ref chain doesn't (or vice versa) — the Writer action
		// helper catches the error mismatch, or the chain growth law
		// catches the replay divergence.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			auditlogtest.AssertAuditLogModel(ft,
				func() auditchain.AuditLog { return auditchain.NewBrokenAuditLog() },
				auditlogtest.AuditLogModelReference(func() auditchain.AuditLog {
					return auditchain.NewInMemoryAuditLog()
				}),
			)
		}()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("negative chain test timed out")
		}
		if !ft.Failed() {
			t.Fatal("framework should have caught dropped entries")
		}
	})

	t.Run("catches tampered hash chain", func(t *testing.T) {
		t.Parallel()
		// Negative: TamperedAuditLog always fails Verify after first append.
		// AUTO-HASH-CHAIN-INTEGRITY fires.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			auditlogtest.AssertAuditLogModel(ft,
				func() auditchain.AuditLog { return auditchain.NewTamperedAuditLog() },
				auditlogtest.AuditLogModelReference(func() auditchain.AuditLog {
					return auditchain.NewInMemoryAuditLog()
				}),
			)
		}()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("negative hash test timed out")
		}
		if !ft.Failed() {
			t.Fatal("framework should have caught tampered hash chain")
		}
	})
}
