// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package predicate is the detector-axis fixture for the predicate shape:
// a question about current state: no arguments, no failure mode. The empty
// parameter list is the signature's distinguishing feature.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package predicate

// Predicate is the fixture interface.
//
//testkit:out predicatetest/ pkg=predicatetest
//testkit:stub
//testkit:suite
type Predicate interface {
	IsEmpty() bool
}
