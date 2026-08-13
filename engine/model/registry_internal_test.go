// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strings"
	"testing"
)

// logRecorder captures the census's once-per-run warning.
type logRecorder struct{ lines []string }

func (l *logRecorder) Logf(format string, args ...any) {
	l.lines = append(l.lines, format)
	_ = args
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
