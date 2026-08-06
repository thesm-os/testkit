// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package voidlifecycle is the detector-axis fixture for the voidlifecycle shape:
// a teardown that cannot fail. With no error return the idempotence law is
// all that remains: calling it twice must be safe.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package voidlifecycle

// VoidLifecycle is the fixture interface.
//
//testkit:out voidlifecycletest/ pkg=voidlifecycletest
//testkit:stub
type VoidLifecycle interface {
	Stop()
}
