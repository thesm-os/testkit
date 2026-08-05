// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package status is the enum-kind fixture: the input the enum generator reads.
//
// The generator derives exhaustiveness, stringer round-trip, parse, and
// out-of-range behaviour from a typed constant block, so the fixture needs a
// gap in the sequence and a value outside the declared range to be meaningful.
package status

import "fmt"

// Status is the enumerated type.
type Status int

// The declared values. Draft is deliberately non-zero so a zero Status is
// invalid rather than merely first, which is what makes out-of-range
// detectable.
const (
	Draft Status = iota + 1
	Published
	Archived
)

// String implements [fmt.Stringer].
func (s Status) String() string {
	switch s {
	case Draft:
		return "draft"
	case Published:
		return "published"
	case Archived:
		return "archived"
	default:
		return fmt.Sprintf("Status(%d)", int(s))
	}
}

// Parse maps a string back to a [Status], which is the round-trip half of the
// stringer law.
func Parse(s string) (Status, error) {
	switch s {
	case "draft":
		return Draft, nil
	case "published":
		return Published, nil
	case "archived":
		return Archived, nil
	default:
		return 0, fmt.Errorf("status: unknown value %q", s)
	}
}
