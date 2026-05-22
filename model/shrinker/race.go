// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shrinker

// RaceEvent is one observed operation on a shared resource during
// a concurrent run.
type RaceEvent struct {
	// Goroutine is the engine-assigned worker ID (NOT the OS
	// runtime.Goroutine ID). Stable across runs for a given seed.
	Goroutine int

	// Op is the operation name (e.g., "Put", "Get"). Reported in
	// the minimized output.
	Op string

	// Resource is the shared resource the operation touched
	// (typically a field name or aggregate id). Two events on the
	// same Resource from different Goroutines race when at least
	// one is a Write.
	Resource string

	// Write reports whether the operation mutated the resource.
	// A read-read pair across goroutines does not race; at least
	// one of the two events must be a Write.
	Write bool
}

// MinimalRacingPair returns the earliest pair of events that
// constitute a data race: distinct goroutines, same resource,
// at least one write. The pair preserves event-order so the first
// event chronologically precedes the second. Returns the zero
// pair plus false when no such pair exists.
func MinimalRacingPair(events []RaceEvent) (RaceEvent, RaceEvent, bool) {
	// A read-only event can still race when a later write touches
	// the same resource from a different goroutine — keep both
	// reads and writes in the scan.
	for i, a := range events {
		for j := i + 1; j < len(events); j++ {
			b := events[j]
			if a.Goroutine == b.Goroutine {
				continue
			}
			if a.Resource != b.Resource {
				continue
			}
			if !a.Write && !b.Write {
				continue
			}
			return a, b, true
		}
	}
	var zero RaceEvent
	return zero, zero, false
}

// MinimizeSchedule iteratively removes events from the sequence
// and keeps the shortest sub-sequence that the consumer-supplied
// probe still reports as racing (probe returns true). Implements
// a simple delta-debugging loop: repeatedly try to drop each
// event; when dropping succeeds, restart from the smaller
// sequence. Terminates when no single-event drop preserves the
// race.
//
// probe must be deterministic: same input must always produce the
// same answer. The probe is consulted at every shrink step.
//
// Returns the minimal racing sub-sequence. If probe initially
// reports false, returns the original events unchanged.
func MinimizeSchedule(events []RaceEvent, probe func([]RaceEvent) bool) []RaceEvent {
	if !probe(events) {
		return events
	}
	current := append([]RaceEvent(nil), events...)
	for shrunk := true; shrunk; {
		shrunk = false
		for i := range current {
			candidate := make([]RaceEvent, 0, len(current)-1)
			candidate = append(candidate, current[:i]...)
			candidate = append(candidate, current[i+1:]...)
			if probe(candidate) {
				current = candidate
				shrunk = true
				break
			}
		}
	}
	return current
}
