// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package embedded is the language-axis fixture for an interface embedding another, so the method set is larger than the
// declaration.
//
// This axis varies the Go type system rather than the classification, so
// these break generators independently of any directive.
package embedded

import (
	"context"
)

// Base is embedded into [Composed].
type Base interface {
	Ping(ctx context.Context) error
}

// Closer is embedded into [Composed] alongside [Base].
type Closer interface {
	Close(ctx context.Context) error
}

// Composed embeds two interfaces and adds one method. A generator that reads
// only the declared methods emits an incomplete stub, which fails the
// compile-time interface check.
type Composed interface {
	Base
	Closer
	Get(ctx context.Context, key string) (string, error)
}
