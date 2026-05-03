// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"errors"
	"fmt"
)

//go:generate testkit enum -o enum.gen_test.go Status Priority

// Status represents an item status. Has a stringer.
type Status int

const (
	StatusPending Status = iota // Pending
	StatusActive                // Active
	StatusClosed                // Closed
)

// String implements fmt.Stringer.
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

// ParseStatus parses a string into a Status.
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

// MarshalText implements encoding.TextMarshaler.
func (s Status) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *Status) UnmarshalText(data []byte) error {
	v, err := ParseStatus(string(data))
	if err != nil {
		return err
	}
	*s = v
	return nil
}

// MarshalJSON implements json.Marshaler.
func (s Status) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler.
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

// Priority represents task priority. No stringer.
type Priority int

const (
	PriorityLow    Priority = iota // Low
	PriorityMedium                 // Medium
	PriorityHigh                   // High
)
