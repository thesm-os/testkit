// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package generic is the language-axis fixture for type parameters on the interface itself, so every emitted type has to
// carry them through.
//
// This axis varies the Go type system rather than the classification, so
// these break generators independently of any directive.
package generic

import (
	"context"
)

// Store is generic over key and value. A generator that drops the type
// parameters produces code that does not compile, which is the failure this
// fixture exists to catch.
//
//testkit:out generictest/ pkg=generictest
//testkit:stub
//testkit:suite
//testkit:model witness=string,int
type Store[K comparable, V any] interface {
	Get(ctx context.Context, key K) (V, error)
	Put(ctx context.Context, key K, value V) error
}
