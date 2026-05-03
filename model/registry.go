// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

// Registry holds a set of [law.Law] instances for an interface T.
// The generator populates it with auto-derived laws; consumers add
// custom laws and skip auto-derived ones by ID.
type Registry[T any] struct {
	laws  []law.Law[T]
	ran   map[string]int // ID → times Check ran
	fired map[string]int // ID → times Check returned non-nil (violations)
}

// NewRegistry creates an empty [Registry].
func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{
		ran:   make(map[string]int),
		fired: make(map[string]int),
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
		if err := l.Check(rt, sut, ref); err != nil {
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
	for k, v := range r.ran {
		ranCp[k] = v
	}
	firedCp := make(map[string]int, len(r.fired))
	for k, v := range r.fired {
		firedCp[k] = v
	}
	return ranCp, firedCp
}
