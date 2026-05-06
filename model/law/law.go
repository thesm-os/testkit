// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package law defines type-parametric algebraic invariants for
// property-based state-machine testing.
//
// Each [Law] encodes a single algebraic property that is checked
// observationally — laws read from the SUT and reference but never
// write. State mutation happens in [model.Action] handlers; laws
// verify that the mutation was correct.
//
// Laws receive [rapid.T] to draw fresh samples per check, integrating
// with rapid's shrinking and generation.
package law

import (
	"errors"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"
)

// Law is a type-parametric invariant checked after every action.
//
// CONTRACT: Laws must be observational. They may read from the SUT
// and reference but must NEVER mutate state. State mutation belongs
// exclusively in [model.Action] handlers. This separation ensures
// laws don't inject invisible operations between commands, don't
// pollute state across iterations, and are safe for mutation testing
// (Pillar 8) where laws validate that synthetic bugs are detectable.
//
// Check receives [rapid.T] so laws can draw fresh samples per
// invocation, integrating with rapid's shrinking.
type Law[T any] interface {
	// ID returns a stable identifier for this law.
	ID() string

	// REQID returns a requirement tag (e.g., "REQ-PKG-FOO-001").
	// Empty for auto-derived laws unless tagged.
	REQID() string

	// Check verifies the law holds. Must not mutate sut or ref.
	Check(rt *rapid.T, sut, ref T) error
}

// StatefulLaw extends [Law] with access to the current step count.
// Use this for laws that need cross-action state tracking, such as
// chain-growth monotonicity (AppendOnlyHistoryGrows) or
// frozen-after-poison invariants. The runner detects StatefulLaw
// via interface assertion and passes the step number.
type StatefulLaw[T any] interface {
	Law[T]

	// CheckWithStep is called instead of Check when the law implements
	// StatefulLaw. Step is the 0-based action count within the current
	// rapid iteration.
	CheckWithStep(rt *rapid.T, sut, ref T, step int) error
}


// ReadAfterWrite checks that every key in a sample pool is consistent
// between SUT and reference. Observational — never writes.
//
// The generator populates Keys with the same pool the Put/Get actions
// draw from. For any key, SUT.Read(key) must equal ref.Read(key).
type ReadAfterWrite[T any, K comparable, V any] struct {
	Read func(*rapid.T, T, K) (V, error)
	Keys *rapid.Generator[K]
}

// ID returns the stable identifier for this law.
func (ReadAfterWrite[T, K, V]) ID() string { return "AUTO-READ-AFTER-WRITE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (ReadAfterWrite[T, K, V]) REQID() string { return "" }

// Check verifies the law holds for the given SUT and reference.
func (l ReadAfterWrite[T, K, V]) Check(rt *rapid.T, sut, ref T) error {
	k := l.Keys.Draw(rt, "ReadAfterWrite_key")
	sutGot, sutErr := l.Read(rt, sut, k)
	refGot, refErr := l.Read(rt, ref, k)
	if (sutErr == nil) != (refErr == nil) {
		//nolint:errorlint // diagnostic message, not wrapping
		return fmt.Errorf("ReadAfterWrite: key %v: SUT err=%v, ref err=%v",
			k, sutErr, refErr)
	}
	if sutErr != nil {
		return nil //nolint:nilerr // both errored — agreement, not a bug
	}
	if diff := cmp.Diff(refGot, sutGot); diff != "" {
		return fmt.Errorf("ReadAfterWrite: key %v: SUT/ref disagree (-ref +sut):\n%s", k, diff)
	}
	return nil
}

// DeleteReturnsNotFound checks that where the reference returns the
// sentinel error, the SUT also returns it. Observational — never writes.
type DeleteReturnsNotFound[T any, K comparable, V any] struct {
	Read     func(*rapid.T, T, K) (V, error)
	Keys     *rapid.Generator[K]
	Sentinel error
}

// ID returns the stable identifier for this law.
func (DeleteReturnsNotFound[T, K, V]) ID() string { return "AUTO-DELETE-RETURNS-NOT-FOUND" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (DeleteReturnsNotFound[T, K, V]) REQID() string { return "" }

// Check verifies the law holds for the given SUT and reference.
func (l DeleteReturnsNotFound[T, K, V]) Check(rt *rapid.T, sut, ref T) error {
	k := l.Keys.Draw(rt, "DeleteReturnsNotFound_key")
	_, refErr := l.Read(rt, ref, k)
	if !errors.Is(refErr, l.Sentinel) {
		return nil // ref says key exists; skip
	}
	_, sutErr := l.Read(rt, sut, k)
	if !errors.Is(sutErr, l.Sentinel) {
		//nolint:errorlint // diagnostic message, not wrapping
		return fmt.Errorf(
			"DeleteReturnsNotFound: key %v: ref returned sentinel %v but SUT returned %v",
			k, l.Sentinel, sutErr)
	}
	return nil
}

// CountEqualsReference checks that the SUT's count matches the
// reference's count. Purely observational.
type CountEqualsReference[T any, R comparable] struct {
	Count func(*rapid.T, T) (R, error)
}

// ID returns the stable identifier for this law.
func (CountEqualsReference[T, R]) ID() string { return "AUTO-COUNT-EQUALS-REFERENCE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (CountEqualsReference[T, R]) REQID() string { return "" }

// Check verifies the law holds for the given SUT and reference.
func (l CountEqualsReference[T, R]) Check(rt *rapid.T, sut, ref T) error {
	sutN, sutErr := l.Count(rt, sut)
	refN, refErr := l.Count(rt, ref)
	if sutErr != nil || refErr != nil {
		//nolint:errorlint // diagnostic message, not wrapping
		return fmt.Errorf("CountEqualsReference: SUT err=%v, ref err=%v",
			sutErr, refErr)
	}
	if sutN != refN {
		return fmt.Errorf("CountEqualsReference: SUT=%v, ref=%v", sutN, refN)
	}
	return nil
}
