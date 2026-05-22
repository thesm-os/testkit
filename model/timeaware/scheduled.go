// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // diagnostic, not wrapping
package timeaware

import (
	"fmt"
	"time"

	"pgregory.net/rapid"
)

// ScheduledFiresAfterAdvance verifies that scheduled tasks fire
// when the clock advances past their scheduled offsets. The
// checker schedules N tasks at offsets drawn from the supplied
// generator, advances the clock past the largest offset, and
// asserts the SUT reports exactly N firings.
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
func (ScheduledFiresAfterAdvance[T]) ID() string { return "AUTO-SCHEDULED-FIRES-AFTER-ADVANCE" }

// REQID returns an empty string (auto-derived).
func (ScheduledFiresAfterAdvance[T]) REQID() string { return "" }

// Check schedules N tasks, advances past the longest offset, and
// verifies the fired count equals N.
func (l ScheduledFiresAfterAdvance[T]) Check(rt *rapid.T, sut, _ T) error {
	n := l.N
	if n <= 0 {
		n = 4
	}
	scheduled := 0
	maxOffset := time.Duration(0)
	for range n {
		offset := l.Offsets.Draw(rt, "schedule_offset")
		if err := l.Schedule(rt, sut, offset); err != nil {
			continue
		}
		scheduled++
		if offset > maxOffset {
			maxOffset = offset
		}
	}
	if scheduled == 0 {
		return nil // every Schedule errored; law vacuous
	}

	before := l.FiredCount(rt, sut)
	// Advance with epsilon so the longest offset is strictly past.
	l.Advance(maxOffset + time.Millisecond)
	after := l.FiredCount(rt, sut)

	if delta := after - before; delta != scheduled {
		return fmt.Errorf(
			"scheduled-fires law: scheduled %d, fired %d after advance to %v",
			scheduled, delta, maxOffset+time.Millisecond,
		)
	}
	return nil
}
