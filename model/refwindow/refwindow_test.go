// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package refwindow_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/refwindow"
)

func TestRollingCounter(t *testing.T) {
	t.Parallel()

	t.Run("counts within the window", func(t *testing.T) {
		t.Parallel()
		now := time.Unix(0, 0)
		clock := func() time.Time { return now }
		r := refwindow.NewRollingCounter[string](10*time.Second, clock)
		_ = r.Incr(t.Context(), "k")
		now = now.Add(1 * time.Second)
		_ = r.Incr(t.Context(), "k")
		got, _ := r.Count(t.Context(), "k")
		testkit.Equal(t, got, 2, "both within window")
	})

	t.Run("evicts events outside the window", func(t *testing.T) {
		t.Parallel()
		now := time.Unix(0, 0)
		clock := func() time.Time { return now }
		r := refwindow.NewRollingCounter[string](10*time.Second, clock)
		_ = r.Incr(t.Context(), "k")
		now = now.Add(20 * time.Second)
		_ = r.Incr(t.Context(), "k")
		got, _ := r.Count(t.Context(), "k")
		testkit.Equal(t, got, 1, "first event evicted")
	})

	t.Run("untouched key reports zero", func(t *testing.T) {
		t.Parallel()
		clock := func() time.Time { return time.Unix(0, 0) }
		r := refwindow.NewRollingCounter[string](time.Second, clock)
		got, _ := r.Count(t.Context(), "missing")
		testkit.Equal(t, got, 0, "no events")
	})
}
