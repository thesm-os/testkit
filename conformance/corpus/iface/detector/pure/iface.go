// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pure is the detector-axis fixture for the pure shape:
// a value derived from nothing but the receiver — no arguments, no context,
// no error. Repeated calls must agree, which is the law the shape carries.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package pure

// Pure is the fixture interface.
//
//testkit:out puretest/ pkg=puretest
//testkit:stub
//testkit:suite
type Pure interface {
	Describe() string
}
