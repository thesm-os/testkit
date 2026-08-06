// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package split

import (
	"errors"
	"fmt"
)

// ErrConflict is the write half's sentinel. A generator scanning only the
// first file it found variables in would miss it, and the uniqueness and
// non-overlap checks would then pass over an incomplete set — which is worse
// than not running, because they would report success.
var ErrConflict = errors.New("split: conflict")

// WriteError is a custom error type in a third file again, so the type scan is
// exercised across files as well as the variable scan.
type WriteError struct {
	Op string
}

// Error implements error.
func (e *WriteError) Error() string { return fmt.Sprintf("split: %s failed", e.Op) }
