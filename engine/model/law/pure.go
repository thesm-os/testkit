// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
)

// PureDeterminism checks that a Pure-shaped method returns the same
// result on repeated calls. Pure methods have no side effects, so
// multiple calls in a law are observational by definition.
//
// The action-level check verifies SUT==ref agreement on a single call;
// this law catches non-deterministic SUTs that agree with the ref once
// but diverge on subsequent calls within the same state.
type PureDeterminism[T any, R any] struct {
	// Call invokes the pure method on the given implementation.
	Call func(*rapid.T, T) R

	// N is the number of repetitions. Zero defaults to 3.
	N int
}

// ID returns the stable identifier for this law.
func (PureDeterminism[T, R]) ID() string { return lawid.PureDeterministic }

// REQID returns an empty string (auto-derived law).
func (PureDeterminism[T, R]) REQID() string { return "" }

// Check verifies determinism by calling the pure method N times on
// the SUT and comparing all results against the first.
func (l PureDeterminism[T, R]) Check(rt *rapid.T, sut, _ T) error {
	n := l.N
	if n <= 0 {
		n = 3
	}
	first := l.Call(rt, sut)
	for i := 1; i < n; i++ {
		got := l.Call(rt, sut)
		if diff := cmp.Diff(first, got); diff != "" {
			return fmt.Errorf("PureDeterminism: call %d differs from first (-first +got):\n%s", i+1, diff)
		}
	}
	return nil
}
