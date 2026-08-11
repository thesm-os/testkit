// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package aggregator is the detector-axis fixture for the aggregator shape:
// whole-collection state reduced to one value. Taking no key is the
// distinction from a reader: there is nothing to look up, only something to compute.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package aggregator

import (
	"context"
)

// Aggregator is the fixture interface.
//
//testkit:out aggregatortest/ pkg=aggregatortest
//testkit:stub
//testkit:suite
type Aggregator interface {
	Count(ctx context.Context) (int, error)
}
