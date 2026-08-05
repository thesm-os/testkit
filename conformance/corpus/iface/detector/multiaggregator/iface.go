// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package multiaggregator is the detector-axis fixture for the multiaggregator shape:
// several reductions at once. Returning them together rather than through
// repeated calls is the point: the numbers only mean anything if they describe one instant.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package multiaggregator

import (
	"context"
)

// MultiAggregator is the fixture interface.
type MultiAggregator interface {
	Stats(ctx context.Context) (int, int, error)
}
