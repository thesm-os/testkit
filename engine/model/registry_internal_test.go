// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"strings"
	"testing"
)

// logRecorder captures what the once-per-run reports actually say.
//
// Formatted rather than kept as the template: a recorder holding "%s" can
// tell you a line was emitted and nothing about whether it named the thing
// the reader needs.
type logRecorder struct{ lines []string }

func (l *logRecorder) Logf(format string, args ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

// The vacuity census warns exactly once, and only for a law vacuous on every
// check past the floor: sixty vacuous returns beside one real pass are a
// subject that sometimes refuses, not a binding that asserts nothing.
func TestNoteVacuousWarnsOnceAtTheFloor(t *testing.T) {
	t.Parallel()

	t.Run("an all-vacuous law past the floor warns once", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry[int]()
		rec := &logRecorder{}
		for range vacuityFloor + 50 {
			r.noteVacuous(rec, "LAW-A")
		}
		if len(rec.lines) != 1 {
			t.Fatalf("the census warns exactly once, got %d warnings", len(rec.lines))
		}
		if !strings.Contains(rec.lines[0], "vacuous") {
			t.Fatalf("the warning names the vacuity, got %q", rec.lines[0])
		}
	})

	t.Run("a law under the floor stays quiet", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry[int]()
		rec := &logRecorder{}
		for range vacuityFloor - 1 {
			r.noteVacuous(rec, "LAW-B")
		}
		if len(rec.lines) != 0 {
			t.Fatalf("under the floor is not a census finding, got %d warnings", len(rec.lines))
		}
	})

	t.Run("one engaged check clears the all-vacuous verdict", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry[int]()
		rec := &logRecorder{}
		r.ran["LAW-C"]++ // one real, engaged check before the vacuous run
		for range vacuityFloor + 1 {
			r.noteVacuous(rec, "LAW-C")
		}
		if len(rec.lines) != 0 {
			t.Fatalf("a law that engaged once is not all-vacuous, got %d warnings", len(rec.lines))
		}
	})
}

// TestDeclinedNamesTheDoorOnce holds the unarmed-law report to saying each
// name once and naming what would arm it.
func TestDeclinedNamesTheDoorOnce(t *testing.T) {
	t.Parallel()

	r := NewRegistry[struct{}]()
	r.Declined("AUTO-DEADLINE-RESPECTING", "MixedModelClocked")
	r.Declined("AUTO-TTL-EXPIRY", "MixedModelClocked")

	var rec logRecorder
	r.sayDeclined(&rec)
	r.sayDeclined(&rec)

	// Twice called, once said: rapid repeats the step and this is a fact
	// about the configuration, not the draw.
	if len(rec.lines) != 2 {
		t.Fatalf("two declines, said once each: got %d lines", len(rec.lines))
	}

	// The door, not just the absence. "A law did not run" sends the reader
	// looking; the option name is the fix.
	joined := strings.Join(rec.lines, "\n")
	for _, want := range []string{"AUTO-DEADLINE-RESPECTING", "AUTO-TTL-EXPIRY", "MixedModelClocked"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("the report must name %q, got: %s", want, joined)
		}
	}
}
