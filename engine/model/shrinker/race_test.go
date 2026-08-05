// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shrinker_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/shrinker"
)

func TestMinimalRacingPair(t *testing.T) {
	t.Parallel()

	t.Run("write-write pair on the same resource races", func(t *testing.T) {
		t.Parallel()
		events := []shrinker.RaceEvent{
			{Goroutine: 1, Op: "Put", Resource: "k", Write: true},
			{Goroutine: 2, Op: "Put", Resource: "k", Write: true},
		}
		a, b, ok := shrinker.MinimalRacingPair(events)
		testkit.True(t, ok, "racing pair detected")
		testkit.Equal(t, a.Goroutine, 1, "first goroutine")
		testkit.Equal(t, b.Goroutine, 2, "second goroutine")
	})

	t.Run("read-write pair on the same resource races", func(t *testing.T) {
		t.Parallel()
		events := []shrinker.RaceEvent{
			{Goroutine: 1, Op: "Get", Resource: "k", Write: false},
			{Goroutine: 2, Op: "Put", Resource: "k", Write: true},
		}
		_, _, ok := shrinker.MinimalRacingPair(events)
		testkit.True(t, ok, "read-write race")
	})

	t.Run("read-read pair does not race", func(t *testing.T) {
		t.Parallel()
		events := []shrinker.RaceEvent{
			{Goroutine: 1, Op: "Get", Resource: "k", Write: false},
			{Goroutine: 2, Op: "Get", Resource: "k", Write: false},
		}
		_, _, ok := shrinker.MinimalRacingPair(events)
		testkit.False(t, ok, "two reads do not race")
	})

	t.Run("same goroutine on same resource does not race", func(t *testing.T) {
		t.Parallel()
		events := []shrinker.RaceEvent{
			{Goroutine: 1, Op: "Put", Resource: "k", Write: true},
			{Goroutine: 1, Op: "Put", Resource: "k", Write: true},
		}
		_, _, ok := shrinker.MinimalRacingPair(events)
		testkit.False(t, ok, "same goroutine cannot race itself")
	})

	t.Run("different resources do not race", func(t *testing.T) {
		t.Parallel()
		events := []shrinker.RaceEvent{
			{Goroutine: 1, Op: "Put", Resource: "k1", Write: true},
			{Goroutine: 2, Op: "Put", Resource: "k2", Write: true},
		}
		_, _, ok := shrinker.MinimalRacingPair(events)
		testkit.False(t, ok, "disjoint resources")
	})

	t.Run("earliest pair returned even with later pairs available", func(t *testing.T) {
		t.Parallel()
		events := []shrinker.RaceEvent{
			{Goroutine: 1, Op: "noise", Resource: "k1", Write: false},
			{Goroutine: 2, Op: "Put", Resource: "k", Write: true},
			{Goroutine: 1, Op: "Put", Resource: "k", Write: true},
			{Goroutine: 3, Op: "Put", Resource: "k", Write: true},
		}
		a, b, ok := shrinker.MinimalRacingPair(events)
		testkit.True(t, ok, "race found")
		testkit.Equal(t, a.Goroutine, 2, "earliest write")
		testkit.Equal(t, b.Goroutine, 1, "next conflicting event")
	})
}

func TestMinimizeSchedule(t *testing.T) {
	t.Parallel()

	t.Run("shrinks to the minimal racing pair", func(t *testing.T) {
		t.Parallel()
		events := []shrinker.RaceEvent{
			{Goroutine: 1, Op: "noise1"},
			{Goroutine: 2, Op: "noise2"},
			{Goroutine: 1, Op: "PutA", Resource: "k", Write: true},
			{Goroutine: 3, Op: "noise3"},
			{Goroutine: 2, Op: "PutB", Resource: "k", Write: true},
			{Goroutine: 4, Op: "noise4"},
		}
		probe := func(es []shrinker.RaceEvent) bool {
			_, _, ok := shrinker.MinimalRacingPair(es)
			return ok
		}
		got := shrinker.MinimizeSchedule(events, probe)
		testkit.Equal(t, len(got), 2, "shrunk to 2 events")
		testkit.Equal(t, got[0].Op, "PutA", "first event")
		testkit.Equal(t, got[1].Op, "PutB", "second event")
	})

	t.Run("probe initially false → unchanged sequence", func(t *testing.T) {
		t.Parallel()
		events := []shrinker.RaceEvent{
			{Goroutine: 1, Op: "x"},
			{Goroutine: 1, Op: "y"},
		}
		got := shrinker.MinimizeSchedule(events, func(_ []shrinker.RaceEvent) bool { return false })
		testkit.Equal(t, len(got), 2, "unchanged")
	})

	t.Run("already-minimal sequence returns as-is", func(t *testing.T) {
		t.Parallel()
		events := []shrinker.RaceEvent{
			{Goroutine: 1, Op: "Put", Resource: "k", Write: true},
			{Goroutine: 2, Op: "Put", Resource: "k", Write: true},
		}
		got := shrinker.MinimizeSchedule(events, func(es []shrinker.RaceEvent) bool {
			_, _, ok := shrinker.MinimalRacingPair(es)
			return ok
		})
		testkit.Equal(t, len(got), 2, "already minimal")
	})
}
