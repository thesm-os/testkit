// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"maps"
	"slices"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
)

// Registry holds a set of [law.Law] instances for an interface T.
// The generator populates it with auto-derived laws; consumers add
// custom laws and skip auto-derived ones by ID.
type Registry[T any] struct {
	laws    []law.Law[T]
	ran     map[string]int // ID → times Check ran
	fired   map[string]int // ID → times Check returned non-nil (violations)
	vacuous map[string]int // ID → times Check returned law.Vacuous
	warned  map[string]bool

	// declined names the laws the bindings selected and could not arm, each
	// with the option that would.
	//
	// Recorded rather than dropped, because the alternative is what this
	// field exists to end: a law behind an unsupplied door was simply never
	// added, so the run asserted one thing fewer than the interface declared
	// and nothing anywhere said which. The generated header names the door;
	// a header is read once and a run is read every time.
	declined     []string
	saidDeclined bool
}

// NewRegistry creates an empty [Registry].
func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{
		ran:     make(map[string]int),
		fired:   make(map[string]int),
		vacuous: make(map[string]int),
		warned:  make(map[string]bool),
	}
}

// vacuityFloor is how many consecutive all-vacuous checks a law is allowed
// before the run says its name: below it, early draws simply have not
// reached the claim yet; at it, the binding is reading as coverage while
// checking nothing.
const vacuityFloor = 200

// noteVacuous records a check the subject declined, and says a law's name
// once per run when every single check so far declined — the not-applicable
// census RFC-0003 commissions, surfaced where a reader is already looking.
func (r *Registry[T]) noteVacuous(rt interface{ Logf(string, ...any) }, id string) {
	r.ran[id]++
	r.vacuous[id]++
	if !r.warned[id] && r.vacuous[id] >= vacuityFloor && r.vacuous[id] == r.ran[id] {
		r.warned[id] = true
		rt.Logf("law %s has been vacuous on all %d checks — the subject refuses "+
			"every precondition this run supplies, and the binding is asserting "+
			"nothing; widen the pools or supply accepted values", id, r.vacuous[id])
	}
}

// Declined records a law that was selected for this interface and could not be
// armed, naming the option that arms it.
//
// The absence is the point. A law behind a door the consumer did not open is
// indistinguishable, from inside the run, from a law nobody selected — and the
// two mean opposite things: one is a claim not made, the other a claim made
// and not checked.
//
// Where this reaches, and where it does not. The message goes to the rapid T,
// which shows it on a failing run and under -v, and swallows it on a quiet
// green one — so this is the channel for a reader already looking, not the
// one that goes and finds them. The generated header is that: it marks each
// conditional law and names the option beside it, and it is the only record
// that survives a run nobody watched. Both exist because they answer at
// different moments, and neither is a substitute for the other.
func (r *Registry[T]) Declined(id, option string) {
	r.declined = append(r.declined, "law "+id+" was selected for this interface and is not "+
		"registered: it needs "+option+", which this run did not supply")
}

// sayDeclined names every unarmed law once per run.
//
// Once, at the first step: rapid repeats CheckAll and the message is about
// the configuration rather than the draw, so repeating it per iteration would
// bury the run's real output under a constant.
//
// Takes the same narrow interface [Registry.noteVacuous] does, so the census
// and this share a channel and a test can hold both to it.
func (r *Registry[T]) sayDeclined(rt interface{ Logf(string, ...any) }) {
	if r.saidDeclined {
		return
	}
	r.saidDeclined = true
	for _, d := range r.declined {
		rt.Logf("%s", d)
	}
}

// Add appends a law to the registry.
func (r *Registry[T]) Add(l law.Law[T]) {
	r.laws = append(r.laws, l)
}

// SkipByID removes the law with the given ID. Returns true if a law
// was removed, false if no law matched (indicating a likely typo).
func (r *Registry[T]) SkipByID(id string) bool {
	before := len(r.laws)
	r.laws = slices.DeleteFunc(r.laws, func(l law.Law[T]) bool {
		return l.ID() == id
	})
	return len(r.laws) < before
}

// Laws returns a defensive copy of all registered laws.
func (r *Registry[T]) Laws() []law.Law[T] {
	return slices.Clone(r.laws)
}

// CheckAll runs every registered law. Returns the first error.
func (r *Registry[T]) CheckAll(rt *rapid.T, sut, ref T) error {
	r.sayDeclined(rt)
	for _, l := range r.laws {
		r.ran[l.ID()]++
		err := l.Check(rt, sut, ref)
		if err != nil {
			r.fired[l.ID()]++
			return err
		}
	}
	return nil
}

// Coverage returns (ran, fired) counts per law ID. Use to detect:
//   - Laws that never ran (misconfigured).
//   - Laws that ran but never fired (possibly too weak).
//   - Laws that always fired (possibly too strict or broken SUT).
func (r *Registry[T]) Coverage() (ran, fired map[string]int) {
	ranCp := make(map[string]int, len(r.ran))
	maps.Copy(ranCp, r.ran)
	firedCp := make(map[string]int, len(r.fired))
	maps.Copy(firedCp, r.fired)
	return ranCp, firedCp
}
