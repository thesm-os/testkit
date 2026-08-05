// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package variadic is the language-axis fixture for a trailing variadic parameter, which changes both the call-forwarding
// and the recorded-call shape.
//
// This axis varies the Go type system rather than the classification, so
// these break generators independently of any directive.
package variadic

import (
	"context"
)

// Finder takes a variadic key list. Forwarding it needs the ellipsis at the
// call site, and recording it needs a slice field rather than a scalar.
type Finder interface {
	Find(ctx context.Context, keys ...string) ([]string, error)
	FindWithLimit(ctx context.Context, limit int, keys ...string) ([]string, error)
}
