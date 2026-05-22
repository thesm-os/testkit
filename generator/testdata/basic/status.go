// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"errors"
	"fmt"
)

//go:generate testkit enum -o status.gen_test.go Status

// Status is a fully-equipped enum: stringer, ParseStatus, and
// encoding.TextMarshaler / json.Marshaler implementations. Used by
// the enum generator to exercise every conditional branch of the
// emitted test file in a single fixture type.
type Status int

// Status values. Inline comments are the canonical "expected
// stringer output" — the enum generator picks them up via
// [generator.Package.Const]'s Comment field.
const (
	StatusPending Status = iota // Pending
	StatusActive                // Active
	StatusClosed                // Closed
)

// String renders the Status as its canonical name. Out-of-range
// values fall back to a `Status(N)` form so the enum generator's
// boundary subtest has something to assert.
func (s Status) String() string {
	switch s {
	case StatusPending:
		return "Pending"
	case StatusActive:
		return "Active"
	case StatusClosed:
		return "Closed"
	default:
		return fmt.Sprintf("Status(%d)", int(s))
	}
}

// ParseStatus is the inverse of String. Returns a typed sentinel
// error for unknown inputs so callers can distinguish parse failures
// from wrapped errors.
func ParseStatus(s string) (Status, error) {
	switch s {
	case "Pending":
		return StatusPending, nil
	case "Active":
		return StatusActive, nil
	case "Closed":
		return StatusClosed, nil
	default:
		return 0, errors.New("unknown status: " + s)
	}
}

// MarshalText delegates to String for encoding.TextMarshaler.
func (s Status) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText delegates to ParseStatus for encoding.TextUnmarshaler.
func (s *Status) UnmarshalText(data []byte) error {
	v, err := ParseStatus(string(data))
	if err != nil {
		return err
	}
	*s = v
	return nil
}

// MarshalBinary encodes the Status as a single byte. Used by the
// enum generator's binary-marshal subtest to verify encoding round-trips
// for enums persisted to durable storage.
func (s Status) MarshalBinary() ([]byte, error) {
	return []byte{byte(s)}, nil
}

// UnmarshalBinary decodes a single byte back into a Status.
func (s *Status) UnmarshalBinary(data []byte) error {
	if len(data) != 1 {
		return errors.New("basic: Status.UnmarshalBinary: expected exactly 1 byte")
	}
	*s = Status(data[0])
	return nil
}

// MarshalJSON quotes the stringer output so JSON consumers get a
// human-readable enum value.
func (s Status) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON strips JSON-string quoting, then delegates to
// ParseStatus.
func (s *Status) UnmarshalJSON(data []byte) error {
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	v, err := ParseStatus(str)
	if err != nil {
		return err
	}
	*s = v
	return nil
}
