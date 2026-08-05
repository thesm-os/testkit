// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/ref"
)

func TestPureScheduler(t *testing.T) {
	t.Parallel()

	stringLess := func(a, b string) bool { return a < b }
	identity := func(n string) string { return n }

	t.Run("linear chain schedules in dependency order", func(t *testing.T) {
		t.Parallel()
		s := ref.NewPureScheduler[string, string](stringLess, identity)
		// a → b → c (a must run first)
		out, err := s.Schedule([]string{"a", "b", "c"}, map[string][]string{
			"b": {"a"},
			"c": {"b"},
		})
		testkit.NoError(t, err, "schedule")
		testkit.Equal(t, out, []string{"a", "b", "c"}, "ordered")
	})

	t.Run("ties broken deterministically by less", func(t *testing.T) {
		t.Parallel()
		s := ref.NewPureScheduler[string, string](stringLess, identity)
		// b and c both depend on a; alphabetical tie-break puts b before c.
		out, _ := s.Schedule([]string{"a", "b", "c"}, map[string][]string{
			"b": {"a"},
			"c": {"a"},
		})
		testkit.Equal(t, out, []string{"a", "b", "c"}, "alphabetical tiebreak")
	})

	t.Run("cycle detection", func(t *testing.T) {
		t.Parallel()
		s := ref.NewPureScheduler[string, string](stringLess, identity)
		_, err := s.Schedule([]string{"a", "b"}, map[string][]string{
			"a": {"b"},
			"b": {"a"},
		})
		testkit.True(t, errors.Is(err, ref.ErrCycle), "cycle detected")
	})

	t.Run("empty graph yields empty schedule", func(t *testing.T) {
		t.Parallel()
		s := ref.NewPureScheduler[string, string](stringLess, identity)
		out, err := s.Schedule(nil, nil)
		testkit.NoError(t, err, "empty")
		testkit.Equal(t, len(out), 0, "empty result")
	})
}
