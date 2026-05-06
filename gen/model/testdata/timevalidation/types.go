// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package timevalidation contains interfaces that violate time-aware
// codegen constraints. Used by generator_test.go to exercise
// validateTimeAware error paths.
package timevalidation

import "context"

// TimeAwareNoReader has //testkit:time-aware but no Reader-shaped
// method. Clock advancement has no observable effect.
type TimeAwareNoReader interface {
	//testkit:time-aware
	Close(ctx context.Context) error
}
