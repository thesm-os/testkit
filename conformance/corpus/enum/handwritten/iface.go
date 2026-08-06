// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package handwritten is the fixture for a type that already has the surface.
//
// An author who wrote their own String meant to keep it, so it must not be
// regenerated — a second declaration is a redeclaration error in their own
// package, against a file they did not write.
//
// Parse rides out with it. Parse and Values are package-level rather than
// methods, so a same-named declaration is invisible to the enum node; a
// generator emitting them anyway would shadow whatever the author wrote. The
// conservative reading is that a type keeping its own String keeps its own
// Parse, which this fixture declares to prove the pair travels together.
//
// UnmarshalText goes with them, because it is written in terms of Parse. What
// remains is IsValid and MarshalText, which depend on neither.
package handwritten

// Weekday is the enumerated type.
//
//testkit:enum
type Weekday int

// The declared values.
const (
	Monday Weekday = iota + 1
	Tuesday
	Wednesday
)

// String is deliberately hand-written, and deliberately not what the generator
// would have derived: lowercase, where the derivation would give `Monday`. A
// fixture whose hand-written form matched the derived one could not tell "the
// author's was kept" from "the generator's was emitted".
func (w Weekday) String() string {
	switch w {
	case Monday:
		return "mon"
	case Tuesday:
		return "tue"
	case Wednesday:
		return "wed"
	}
	return "unknown"
}

// ParseWeekday is the author's own, matching their String.
func ParseWeekday(s string) (Weekday, error) {
	for _, w := range []Weekday{Monday, Tuesday, Wednesday} {
		if w.String() == s {
			return w, nil
		}
	}
	return 0, ErrUnknownWeekday
}
