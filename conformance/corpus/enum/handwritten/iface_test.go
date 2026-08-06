// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package handwritten_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/enum/handwritten"
)

// The generated checks cannot reach this package's own String and
// ParseWeekday: the generator declines to emit a surface the author already
// wrote, and therefore also declines to assert anything about it. That is the
// right call — it cannot know what the author's version promises — but it
// leaves the fixture's hand-written half unexercised unless something here
// covers it.
//
// So these are the checks a consumer would write for their own implementation,
// and the fixture is the worked example of writing them: what the generator
// steps back from, the author picks up.
func TestWeekdayString(t *testing.T) {
	t.Parallel()

	t.Run("renders each variant in the author's own spelling", func(t *testing.T) {
		t.Parallel()
		// Deliberately lowercase, where the generator's derivation would give
		// `Monday`. That difference is what proves the hand-written one
		// survived rather than being quietly replaced.
		for _, c := range []struct {
			day  handwritten.Weekday
			want string
		}{
			{handwritten.Monday, "mon"},
			{handwritten.Tuesday, "tue"},
			{handwritten.Wednesday, "wed"},
		} {
			testkit.Equal(t, c.day.String(), c.want, "each variant renders as the author wrote it")
		}
	})

	t.Run("renders an undeclared value distinctly", func(t *testing.T) {
		t.Parallel()
		// A conversion admits any int. A fallback colliding with a declared
		// variant's text would make a corrupt value indistinguishable from a
		// good one in a log.
		out := handwritten.Weekday(99)
		for _, day := range []handwritten.Weekday{
			handwritten.Monday, handwritten.Tuesday, handwritten.Wednesday,
		} {
			testkit.NotEqual(t, out.String(), day.String(), "the fallback must not read as a variant")
		}
	})
}

// ParseWeekday is the author's own, and rides out of generation with String —
// so nothing generated asserts it inverts String, which is the one property it
// exists to have.
func TestParseWeekday(t *testing.T) {
	t.Parallel()

	t.Run("inverts String for every variant", func(t *testing.T) {
		t.Parallel()
		// A value that renders one way and parses another leaves the process
		// and does not come home.
		for _, day := range []handwritten.Weekday{
			handwritten.Monday, handwritten.Tuesday, handwritten.Wednesday,
		} {
			got, err := handwritten.ParseWeekday(day.String())
			testkit.NoError(t, err, "a declared variant must parse back")
			testkit.Equal(t, got, day, "the round trip must be lossless")
		}
	})

	t.Run("refuses text naming no variant", func(t *testing.T) {
		t.Parallel()
		// Admitting it would let an unnamed value in through the one door that
		// exists to keep it out.
		_, err := handwritten.ParseWeekday("not-a-day")
		testkit.ErrorIs(t, err, handwritten.ErrUnknownWeekday, "unknown text must be refused")
	})

	t.Run("reports the zero value for text it refuses", func(t *testing.T) {
		t.Parallel()
		// A caller that ignores the error gets a value the type itself rejects,
		// rather than a plausible-looking variant.
		got, _ := handwritten.ParseWeekday("not-a-day")
		testkit.False(t, got.IsValid(), "a refused parse yields no valid variant")
	})
}
