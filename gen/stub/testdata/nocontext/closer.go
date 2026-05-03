// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nocontext

//go:generate testkit stub -o closertest/closer_stub.gen.go Closer

// Closer exercises methods without context parameters.
type Closer interface {
	// Close releases resources. No context, just error.
	Close() error
	// String returns a description. No context, no error.
	String() string
}
