// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package nocontext is the language-axis fixture for methods without a leading context, which every context-aware assertion
// has to skip rather than mis-apply.
//
// This axis varies the Go type system rather than the classification, so
// these break generators independently of any directive.
package nocontext

import (
	"errors"
)

// ErrDivideByZero is reported for a zero divisor.
var ErrDivideByZero = errors.New("nocontext: divide by zero")

// Calculator takes no context anywhere. Shapes are defined with the leading
// context optional, so the same detectors must fire here as on the
// context-carrying forms.
type Calculator interface {
	Add(a, b int) int
	Divide(a, b int) (int, error)
	Reset()
}
