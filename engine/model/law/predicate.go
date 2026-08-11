// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law

import (
	"fmt"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
)

// PredicateConsistency checks that a Predicate-shaped method returns
// the same bool on repeated calls. Predicates have no side effects, so
// multiple calls in a law are observational by definition.
type PredicateConsistency[T any] struct {
	// Call invokes the predicate method on the given implementation.
	Call func(*rapid.T, T) bool

	// N is the number of repetitions. Zero defaults to 3.
	N int
}

// ID returns the stable identifier for this law.
func (PredicateConsistency[T]) ID() string { return lawid.PredicateConsistent }

// REQID returns an empty string (auto-derived law).
func (PredicateConsistency[T]) REQID() string { return "" }

// Check verifies consistency by calling the predicate N times on
// the SUT and comparing all results against the first.
func (l PredicateConsistency[T]) Check(rt *rapid.T, sut, _ T) error {
	n := l.N
	if n <= 0 {
		n = 3
	}
	first := l.Call(rt, sut)
	for i := 1; i < n; i++ {
		got := l.Call(rt, sut)
		if got != first {
			return fmt.Errorf("PredicateConsistency: call %d returned %v, first returned %v", i+1, got, first)
		}
	}
	return nil
}
