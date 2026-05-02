// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package enum is a test fixture for the enum generator.
package enum

import "fmt"

//go:generate stringer -type=Status

// Status represents an order status.
type Status int

const (
	StatusUnspecified Status = iota // Unspecified
	StatusPending                   // Pending
	StatusActive                    // Active
	StatusClosed                    // Closed
)

// String implements fmt.Stringer. In real code this would be generated
// by stringer, but we hand-write it for the test fixture.
func (s Status) String() string {
	switch s {
	case StatusUnspecified:
		return "Unspecified"
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

// Priority is a second enum type to test multi-type generation.
type Priority int

const (
	PriorityLow    Priority = iota // Low
	PriorityMedium                 // Medium
	PriorityHigh                   // High
)

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "Low"
	case PriorityMedium:
		return "Medium"
	case PriorityHigh:
		return "High"
	default:
		return fmt.Sprintf("Priority(%d)", int(p))
	}
}

// Region has no inline comments — String() returns the constant name.
type Region int

const (
	RegionUS Region = iota
	RegionEU
	RegionAP
)

func (r Region) String() string {
	switch r {
	case RegionUS:
		return "RegionUS"
	case RegionEU:
		return "RegionEU"
	case RegionAP:
		return "RegionAP"
	default:
		return fmt.Sprintf("Region(%d)", int(r))
	}
}
