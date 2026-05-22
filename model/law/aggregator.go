// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"fmt"

	"pgregory.net/rapid"
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
func (AggregatorBounded[T, R]) ID() string { return "AUTO-AGGREGATOR-BOUNDED" }

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
func (Associative[T, V, Obs]) ID() string { return "AUTO-ASSOCIATIVE" }

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
func (Conservative[T, V]) ID() string { return "AUTO-CONSERVATIVE" }

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
