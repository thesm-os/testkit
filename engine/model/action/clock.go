// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action

import (
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/timeaware"
)

// AdvanceClock creates an action that advances both SUT and ref
// TestClocks by the same random duration. The clocks getter returns
// the per-iteration clock instances (created fresh each iteration by
// the clock factory).
//
// This action is emitted by the generator when //testkit:time-aware
// is present. It exercises time-dependent behavior deterministically:
// TTL expiry, scheduled fires, deadline compliance.
//
// For concurrent-mode runs, use [AdvanceClockSynced] to coordinate
// the advance with in-flight operations via a [timeaware.Barrier].
func AdvanceClock[T any](
	name string,
	clocks func() (sut, ref *clock.TestClock),
	maxAdvance time.Duration,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, _, _ T) model.ActionResult {
			sutClk, refClk := clocks()
			if sutClk == nil || refClk == nil {
				return model.ActionResult{}
			}
			d := time.Duration(rapid.Int64Range(0, int64(maxAdvance)).Draw(rt, name+"_duration"))
			sutClk.Advance(d)
			refClk.Advance(d)
			return model.ActionResult{}
		},
	}
}

// AdvanceClockSynced is the concurrent-mode counterpart of
// [AdvanceClock]: it runs the dual-clock advance under a
// [timeaware.Barrier] so in-flight operations cannot observe a
// mid-advance state. The barrier's write lock blocks until every
// concurrent op has released; new ops cannot start until the
// advance closure returns.
//
// A nil barrier degrades to [AdvanceClock]'s unsynchronized
// behavior — useful for sequential-mode reuse without a code
// branch in the runner.
func AdvanceClockSynced[T any](
	name string,
	clocks func() (sut, ref *clock.TestClock),
	maxAdvance time.Duration,
	barrier *timeaware.Barrier,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, _, _ T) model.ActionResult {
			sutClk, refClk := clocks()
			if sutClk == nil || refClk == nil {
				return model.ActionResult{}
			}
			d := time.Duration(rapid.Int64Range(0, int64(maxAdvance)).Draw(rt, name+"_duration"))
			advance := func() {
				sutClk.Advance(d)
				refClk.Advance(d)
			}
			if barrier == nil {
				advance()
				return model.ActionResult{}
			}
			barrier.Advance(advance)
			return model.ActionResult{}
		},
	}
}
