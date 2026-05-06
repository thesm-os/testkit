// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"maps"
	"slices"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

// Registry holds a set of [law.Law] and [law.FinalLaw] instances for
// an interface T. The generator populates it with auto-derived laws;
// consumers add custom laws and skip auto-derived ones by ID.
type Registry[T any] struct {
	laws      []law.Law[T]
	finalLaws []law.FinalLaw[T]
	ran       map[string]int // ID → times Check ran
	fired     map[string]int // ID → times Check returned non-nil (violations)
}

// NewRegistry creates an empty [Registry].
func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{
		ran:   make(map[string]int),
		fired: make(map[string]int),
	}
}

// Add appends a law to the registry. The argument must implement
// [law.Law], [law.FinalLaw], or both. Panics if it satisfies neither.
func (r *Registry[T]) Add(l any) {
	matched := false
	if regular, ok := l.(law.Law[T]); ok {
		r.laws = append(r.laws, regular)
		matched = true
	}
	if final, ok := l.(law.FinalLaw[T]); ok {
		r.finalLaws = append(r.finalLaws, final)
		matched = true
	}
	if !matched {
		panic(fmt.Sprintf("model.Registry.Add: argument %T does not implement law.Law or law.FinalLaw", l))
	}
}

// SkipByID removes the law with the given ID from both regular and
// final law lists. Returns true if a law was removed.
func (r *Registry[T]) SkipByID(id string) bool {
	beforeReg := len(r.laws)
	r.laws = slices.DeleteFunc(r.laws, func(l law.Law[T]) bool {
		return l.ID() == id
	})
	beforeFinal := len(r.finalLaws)
	r.finalLaws = slices.DeleteFunc(r.finalLaws, func(l law.FinalLaw[T]) bool {
		return l.ID() == id
	})
	return len(r.laws) < beforeReg || len(r.finalLaws) < beforeFinal
}

// Laws returns a defensive copy of all registered laws.
func (r *Registry[T]) Laws() []law.Law[T] {
	return slices.Clone(r.laws)
}

// FinalLaws returns a defensive copy of all registered final laws.
func (r *Registry[T]) FinalLaws() []law.FinalLaw[T] {
	return slices.Clone(r.finalLaws)
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
