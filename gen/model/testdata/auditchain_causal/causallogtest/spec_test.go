// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package causallogtest_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	ac "go.thesmos.sh/testkit/gen/model/testdata/auditchain_causal"
	"go.thesmos.sh/testkit/gen/model/testdata/auditchain_causal/causallogtest"
)

func TestCausalLogModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 1 causal chain with dep-validating ref", func(t *testing.T) {
		t.Parallel()
		// InMemoryCausalLog validates deps on Append. Tier 1: both SUT
		// and ref validate deps, so random entries with unresolved deps
		// are rejected consistently. Chain laws + ReplayRespectsCausality fire.
		causallogtest.AssertCausalLogModel(t,
			func() ac.CausalLog { return ac.NewInMemoryCausalLog() },
			causallogtest.CausalLogModelReference(func() ac.CausalLog {
				return ac.NewInMemoryCausalLog()
			}),
		)
	})

	t.Run("catches reversed replay via ReplayRespectsCausality", func(t *testing.T) {
		t.Parallel()
		// Negative: ReorderingCausalLog validates deps on Append (so
		// action-level error comparison passes) but returns entries in
		// REVERSE order on Replay. ReplayRespectsCausality detects that
		// deps appear after their dependents in the replay.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			causallogtest.AssertCausalLogModel(ft,
				func() ac.CausalLog { return ac.NewReorderingCausalLog() },
				causallogtest.CausalLogModelReference(func() ac.CausalLog {
					return ac.NewInMemoryCausalLog()
				}),
			)
		}()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("negative causal test timed out")
		}
		if !ft.Failed() {
			t.Fatal("ReplayRespectsCausality should have caught reversed replay")
		}
	})
}
