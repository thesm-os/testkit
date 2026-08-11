// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"fmt"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
)

// AggregatorBounded verifies that an Aggregator-class method's
// numeric result stays within [Min, Max]. Auto-emitted for
// Aggregators carrying //testkit:bounded min..max.
type AggregatorBounded[T any, R interface{ ~int | ~int64 | ~float64 }] struct {
	Read func(*rapid.T, T) (R, error)
	Min  R
	Max  R
}

// ID returns the stable identifier for this law.
func (AggregatorBounded[T, R]) ID() string { return lawid.AggregatorBounded }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (AggregatorBounded[T, R]) REQID() string { return "" }

// Check verifies the result is in [Min, Max].
func (l AggregatorBounded[T, R]) Check(rt *rapid.T, sut, _ T) error {
	got, err := l.Read(rt, sut)
	if err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if got < l.Min || got > l.Max {
		return fmt.Errorf("AggregatorBounded: got %v outside [%v, %v]", got, l.Min, l.Max)
	}
	return nil
}

// Associative verifies that an Aggregator-class binary fold
// associates: (a;b);c == a;(b;c). The law runs both groupings on
// fresh impls and compares the observed result via Observe.
// Auto-emitted for Aggregator methods carrying //testkit:associative.
type Associative[T any, V any, Obs any] struct {
	Factory func() T
	Apply   func(*rapid.T, T, V) error
	Values  *rapid.Generator[V]
	Observe func(*rapid.T, T) Obs
}

// ID returns the stable identifier for this law.
func (Associative[T, V, Obs]) ID() string { return lawid.Associative }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (Associative[T, V, Obs]) REQID() string { return "" }

// Check applies three values in both groupings and compares.
func (l Associative[T, V, Obs]) Check(rt *rapid.T, _, _ T) error {
	a := l.Values.Draw(rt, "Associative_a")
	b := l.Values.Draw(rt, "Associative_b")
	c := l.Values.Draw(rt, "Associative_c")

	left := l.Factory()
	if l.Apply(rt, left, a) != nil || l.Apply(rt, left, b) != nil || l.Apply(rt, left, c) != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}

	right := l.Factory()
	if l.Apply(rt, right, a) != nil || l.Apply(rt, right, b) != nil || l.Apply(rt, right, c) != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}

	leftObs := l.Observe(rt, left)
	rightObs := l.Observe(rt, right)
	if fmt.Sprint(leftObs) != fmt.Sprint(rightObs) {
		return fmt.Errorf("associative law: (%v;%v);%v != %v;(%v;%v): left=%v right=%v",
			a, b, c, a, b, c, leftObs, rightObs)
	}
	return nil
}

// Conservative verifies a Mutator+Aggregator pair preserves the
// sum-of-Field invariant: the sum of the named field before and
// after a mutation are equal. Auto-emitted for Mutator+Aggregator
// pairs carrying //testkit:conservative <Field>.
type Conservative[T any, V any] struct {
	Sum    func(*rapid.T, T) int64 // observe the conserved quantity
	Write  func(*rapid.T, T, V) error
	Values *rapid.Generator[V]
}

// ID returns the stable identifier for this law.
func (Conservative[T, V]) ID() string { return lawid.Conservative }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (Conservative[T, V]) REQID() string { return "" }

// Check verifies Sum(state) is invariant across one Write.
func (l Conservative[T, V]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "Conservative_value")
	before := l.Sum(rt, sut)
	if err := l.Write(rt, sut, v); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	after := l.Sum(rt, sut)
	if before != after {
		return fmt.Errorf("conservative law: value %v: sum changed %d → %d", v, before, after)
	}
	return nil
}

// Windowed verifies that a rolling-window aggregator only reflects
// events within its trailing window: an increment counts immediately
// but decays once the clock advances past the window. Auto-emitted
// for Aggregator/Mutator methods carrying //testkit:windowed
// <Duration>.
//
// Advance moves the aggregator's injected clock forward. The law
// records the count for a key, increments it (which must raise the
// count), advances past Window, and asserts the count drops — a
// counter that never expires entries keeps the increment and fails.
type Windowed[T any, K comparable] struct {
	Incr    func(rt *rapid.T, sut T, k K) error
	Count   func(rt *rapid.T, sut T, k K) (int, error)
	Advance func(d time.Duration)
	Keys    *rapid.Generator[K]
	Window  time.Duration
}

// ID returns the stable identifier for this law.
func (Windowed[T, K]) ID() string { return lawid.Windowed }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (Windowed[T, K]) REQID() string { return "" }

// Check increments a key, confirms the count rose, advances past the
// window, and confirms the increment decayed.
func (l Windowed[T, K]) Check(rt *rapid.T, sut, _ T) error {
	k := l.Keys.Draw(rt, "Windowed_key")
	before, err := l.Count(rt, sut, k)
	if err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if incErr := l.Incr(rt, sut, k); incErr != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	within, err := l.Count(rt, sut, k)
	if err != nil {
		return fmt.Errorf("windowed law: key %v: count after increment errored: %v", k, err)
	}
	if within < before+1 {
		return fmt.Errorf("windowed law: key %v: increment not reflected in count (%d → %d)", k, before, within)
	}
	l.Advance(l.Window + time.Nanosecond)
	after, err := l.Count(rt, sut, k)
	if err != nil {
		return fmt.Errorf("windowed law: key %v: count after advance errored: %v", k, err)
	}
	if after >= within {
		return fmt.Errorf("windowed law: key %v: count did not decay after advancing past the window (%d → %d)",
			k, within, after)
	}
	return nil
}
