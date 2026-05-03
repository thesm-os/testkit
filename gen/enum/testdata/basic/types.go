// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import "fmt"

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

// Priority represents task priority. No stringer.
type Priority int

const (
	PriorityLow    Priority = iota // Low
	PriorityMedium                 // Medium
	PriorityHigh                   // High
)
