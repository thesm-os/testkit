// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"
)

// IdempotentWrite verifies that the second Write of an identical
// value is observably equivalent to the first — repeated writes
// of (key, value) produce the same state. Auto-emitted for
// Writers carrying //testkit:idempotent.
//
// The law performs the comparison via a consumer-supplied probe
// function that reads enough state to detect divergence (typically
// the paired Reader returning V for the same key).
type IdempotentWrite[T any, V any, Obs any] struct {
	Write   func(*rapid.T, T, V) error
	Values  *rapid.Generator[V]
	Observe func(*rapid.T, T) Obs
}

// ID returns the stable identifier for this law.
func (IdempotentWrite[T, V, Obs]) ID() string { return "AUTO-IDEMPOTENT-WRITE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (IdempotentWrite[T, V, Obs]) REQID() string { return "" }

// Check verifies that two identical Writes produce the same Observe.
//
// The law mutates state (it writes twice). Idempotence is a write-
// side property; checking it requires writing — but the second
// Write must not change anything visible to Observe.
func (l IdempotentWrite[T, V, Obs]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "IdempotentWrite_value")
	if err := l.Write(rt, sut, v); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	before := l.Observe(rt, sut)
	if err := l.Write(rt, sut, v); err != nil {
		return fmt.Errorf("IdempotentWrite: value %v: second write errored: %v", v, err)
	}
	after := l.Observe(rt, sut)
	if diff := cmp.Diff(before, after); diff != "" {
		return fmt.Errorf("IdempotentWrite: value %v: second write changed state (-before +after):\n%s", v, diff)
	}
	return nil
}

// CommutativeWrite verifies a;b == b;a observationally over the
// supplied Observe function. Auto-emitted for Mutator/Writer
// methods carrying //testkit:commutative.
//
// The law runs the pair-of-writes on two fresh impls — a;b on one,
// b;a on the other — using the consumer-supplied factory to
// construct them. The result Obs must agree.
type CommutativeWrite[T any, V any, Obs any] struct {
	Factory func() T
	Write   func(*rapid.T, T, V) error
	Values  *rapid.Generator[V]
	Observe func(*rapid.T, T) Obs
}

// ID returns the stable identifier for this law.
func (CommutativeWrite[T, V, Obs]) ID() string { return "AUTO-COMMUTATIVE-WRITE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (CommutativeWrite[T, V, Obs]) REQID() string { return "" }

// Check runs a;b on one impl and b;a on another, comparing Observe.
func (l CommutativeWrite[T, V, Obs]) Check(rt *rapid.T, _, _ T) error {
	a := l.Values.Draw(rt, "CommutativeWrite_a")
	b := l.Values.Draw(rt, "CommutativeWrite_b")

	ab := l.Factory()
	if err := l.Write(rt, ab, a); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if err := l.Write(rt, ab, b); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}

	ba := l.Factory()
	if err := l.Write(rt, ba, b); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if err := l.Write(rt, ba, a); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}

	obsAB := l.Observe(rt, ab)
	obsBA := l.Observe(rt, ba)
	if diff := cmp.Diff(obsAB, obsBA); diff != "" {
		return fmt.Errorf("CommutativeWrite: a=%v b=%v: order matters (-ab +ba):\n%s", a, b, diff)
	}
	return nil
}

// AtomicWrite verifies that a Writer returning an error leaves the
// observable state unchanged — error implies no partial mutation.
// Auto-emitted for Writers carrying //testkit:atomic.
//
// The law snapshots observable state via Observe, calls Write, and
// when Write errors compares the post-error snapshot against the
// pre-call snapshot. Successful writes are skipped (their
// mutation is checked by other laws).
type AtomicWrite[T any, V any, Obs any] struct {
	Write   func(*rapid.T, T, V) error
	Values  *rapid.Generator[V]
	Observe func(*rapid.T, T) Obs
}

// ID returns the stable identifier for this law.
func (AtomicWrite[T, V, Obs]) ID() string { return "AUTO-ATOMIC-WRITE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (AtomicWrite[T, V, Obs]) REQID() string { return "" }

// Check verifies that errored writes leave state unchanged.
func (l AtomicWrite[T, V, Obs]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "AtomicWrite_value")
	before := l.Observe(rt, sut)
	err := l.Write(rt, sut, v)
	if err == nil {
		return nil
	}
	after := l.Observe(rt, sut)
	if diff := cmp.Diff(before, after); diff != "" {
		return fmt.Errorf("AtomicWrite: errored write mutated state (-before +after):\n%s", diff)
	}
	return nil
}

// ValidTransition verifies that Write only advances the named field
// through transitions allowed by a state-machine graph. Auto-
// emitted for Mutator/Writer methods carrying
// //testkit:valid-transition-only <Field>.
//
// The law consults the Allowed predicate to decide whether the
// observed before→after transition was legal; it does not enforce
// the rejection itself (the SUT must reject illegal writes on its
// own). The law only flags an after-state that the predicate
// declares invalid.
type ValidTransition[T any, V any, S comparable] struct {
	Write   func(*rapid.T, T, V) error
	Values  *rapid.Generator[V]
	Observe func(*rapid.T, T) S
	Allowed func(from, to S) bool
}

// ID returns the stable identifier for this law.
func (ValidTransition[T, V, S]) ID() string { return "AUTO-VALID-TRANSITION" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (ValidTransition[T, V, S]) REQID() string { return "" }

// Check verifies any post-write state was reachable from the prior.
func (l ValidTransition[T, V, S]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "ValidTransition_value")
	before := l.Observe(rt, sut)
	if err := l.Write(rt, sut, v); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	after := l.Observe(rt, sut)
	if before == after {
		return nil
	}
	if !l.Allowed(before, after) {
		return fmt.Errorf("ValidTransition: value %v: illegal %v → %v", v, before, after)
	}
	return nil
}
