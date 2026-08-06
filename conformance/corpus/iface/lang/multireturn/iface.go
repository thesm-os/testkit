// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package multireturn is the language-axis fixture for more than two return values, past the point where a generator can assume
// a value-and-error pair.
//
// This axis varies the Go type system rather than the classification, so
// these break generators independently of any directive.
package multireturn

import (
	"context"
)

// Wide returns four values. Anything that assumes the last return is an error
// and the first is the value produces wrong field names here.
//
//testkit:out multireturntest/ pkg=multireturntest
//testkit:stub
type Wide interface {
	Quad(ctx context.Context, id string) (string, int, bool, error)
	Triple(ctx context.Context, id string) (string, int, bool)
	NoError(ctx context.Context, id string) (string, int)
}
