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
