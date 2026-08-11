// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package lifecycle is the detector-axis fixture for the lifecycle shape:
// a failable teardown. Close must be idempotent and every later operation
// must report the closed sentinel, which is the law the shape carries.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package lifecycle

import (
	"context"
	"errors"
)

// ErrClosed is the sentinel operations report after teardown.
var ErrClosed = errors.New("lifecycle: closed")

// Lifecycle is the fixture interface.
//
//testkit:out lifecycletest/ pkg=lifecycletest
//testkit:stub
//testkit:suite
type Lifecycle interface {
	Close(ctx context.Context) error
}
