// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package iterators exercises iter.Seq[T] and iter.Seq2[V, error] return
// types. The generator auto-detects these and emits Yields/YieldsError helpers.
package iterators

import (
	"context"
	"iter"
)

//go:generate testkit stub -o scannertest/scanner_stub.gen.go Scanner

// Item is a scanned value.
type Item struct {
	ID   string
	Data []byte
}

// Scanner exercises iterator return types.
type Scanner interface {
	// Keys returns all keys as a pull iterator.
	Keys(ctx context.Context) iter.Seq[string]

	// Scan returns items with potential error — the common error-yielding pattern.
	Scan(ctx context.Context, prefix string) iter.Seq2[Item, error]
}
