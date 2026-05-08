// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package voidpure exercises void-return methods that have no
// context, no error, and no return value. These must NOT be
// classified as Pure (which requires a return value).
package voidpure

//go:generate testkit suite -o streamtest/stream_spec.gen.go Stream
//go:generate testkit bench -o streamtest/stream_bench.gen.go Stream
//go:generate testkit stub  -o streamtest/stream_stub.gen.go  Stream

// Digest is a sample return type.
type Digest [32]byte

// Stream has a mix of void and typed Pure-shaped methods.
type Stream interface {
	// Sum returns a typed value — Pure shape.
	Sum() Digest

	// Reset has no return — must be Unknown, not Pure.
	Reset()

	// Close has no return — must be Unknown, not Pure.
	Close()
}
