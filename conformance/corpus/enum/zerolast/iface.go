// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package zerolast is the enum-corpus fixture for a set that does not declare
// its zero first.
//
// Declaration order and zero-ness are different questions, and every other
// fixture answers both the same way — so a check that took the zero variant's
// *name* for its message and rebuilt the variant itself by position agreed with
// itself everywhere and was wrong here. The generated assertion read
// `the zero value is Unset` while comparing against `US`, and failed in the
// consumer's repository on a file they did not write.
//
// A string enum, because that is where the shape is ordinary rather than
// contrived: the empty string is the zero, and an author listing real values
// before the absent one is writing the obvious thing.
package zerolast

// Region is the enumerated type.
//
//testkit:enum
type Region string

// The declared values, with the zero last. Nothing about the set requires that
// order, which is the point — it is the order an author would reach for, and
// the one the generator got wrong.
const (
	US    Region = "us-east"
	EU    Region = "eu-west"
	Unset Region = ""
)
