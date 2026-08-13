// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // diagnostic, not wrapping
package timeaware

import (
	"fmt"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/model/law"
)

// ScheduledFiresAfterAdvance verifies that scheduled tasks fire
// when the clock advances past their scheduled offsets. The
// checker schedules N tasks at offsets drawn from the supplied
// generator, advances the clock past the largest offset, and
// asserts the SUT reports at least N new firings.
//
// At least, not exactly: the law shares its subject with an action
// stream that schedules work of its own, and whatever of that is
// pending fires inside the same advance. Exact-count is the
// quiescent-subject claim — the generated fixture's unit tests can
// state it; a law on a shared pair cannot. Whether the pair's fired
// counts agree is CountEqualsReference's claim, not this one's.
type ScheduledFiresAfterAdvance[T any] struct {
	// Schedule registers a task to fire at the given offset from
	// now. Errors abort the law as vacuous.
	Schedule func(rt *rapid.T, sut T, at time.Duration) error

	// FiredCount returns the number of tasks that have fired so
	// far. The checker compares post-advance count to the
	// scheduled count.
	FiredCount func(rt *rapid.T, sut T) int

	// Offsets supplies scheduled-at durations. The checker draws
	// N times; the largest drawn offset becomes the advance
	// target.
	Offsets *rapid.Generator[time.Duration]

	// N is the number of tasks to schedule per Check. Zero
	// defaults to 4.
	N int

	// Advance advances the test clock by the supplied duration.
	Advance func(time.Duration)
}

// ID returns the stable identifier for this law.
func (ScheduledFiresAfterAdvance[T]) ID() string { return lawid.ScheduledFiresAfterAdvance }

// REQID returns an empty string (auto-derived).
func (ScheduledFiresAfterAdvance[T]) REQID() string { return "" }

// Check schedules N tasks, advances past the longest offset, and
// verifies at least N tasks fired.
//
// Every accepted schedule lands on both sides — the mirrored half of the
// law conduct contract: the pair shares one clock, so twins scheduled
// together fire together, and a reference left unscheduled would count a
// divergence the law itself created at the next fired-count comparison.
//
// The fired count is read before any scheduling: a zero-offset task is
// due the instant it registers, and a snapshot taken after would count
// it into the baseline and report the firing as missing.
func (l ScheduledFiresAfterAdvance[T]) Check(rt *rapid.T, sut, ref T) error {
	n := l.N
	if n <= 0 {
		n = 4
	}
	before := l.FiredCount(rt, sut)

	scheduled := 0
	maxOffset := time.Duration(0)
	for range n {
		offset := l.Offsets.Draw(rt, "schedule_offset")
		if err := l.Schedule(rt, sut, offset); err != nil {
			continue
		}
		if err := l.Schedule(rt, ref, offset); err != nil {
			return fmt.Errorf("scheduled-fires law: the reference refused what the subject accepted: %w", err)
		}
		scheduled++
		if offset > maxOffset {
			maxOffset = offset
		}
	}
	if scheduled == 0 {
		return law.Vacuous // every Schedule errored; the claim was never engaged
	}

	// Advance with epsilon so the longest offset is strictly past.
	l.Advance(maxOffset + time.Millisecond)
	after := l.FiredCount(rt, sut)

	if delta := after - before; delta < scheduled {
		return fmt.Errorf(
			"scheduled-fires law: scheduled %d, fired only %d after advance past %v",
			scheduled, delta, maxOffset+time.Millisecond,
		)
	}
	return nil
}
