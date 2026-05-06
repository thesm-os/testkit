// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action

import (
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/model"
)

// AdvanceClock creates an action that advances both SUT and ref
// TestClocks by the same random duration. The clocks getter returns
// the per-iteration clock instances (created fresh each iteration by
// the clock factory).
//
// This action is emitted by the generator when //testkit:time-aware
// is present. It exercises time-dependent behavior deterministically:
// TTL expiry, scheduled fires, deadline compliance.
func AdvanceClock[T any](
	name string,
	clocks func() (sut, ref *clock.TestClock),
	maxAdvance time.Duration,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, _, _ T) {
			sutClk, refClk := clocks()
			if sutClk == nil || refClk == nil {
				return // no clock factory configured
			}
			d := time.Duration(rapid.Int64Range(0, int64(maxAdvance)).Draw(rt, name+"_duration"))
			sutClk.Advance(d)
			refClk.Advance(d)
		},
	}
}
