// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package iterators exercises the spec generator with iter.Seq and
// iter.Seq2 return types. Auto-detected iterator tests include empty
// iteration, break mid-stream, error-free iteration, and double iteration.
package iterators

import (
	"context"
	"iter"
)

//go:generate testkit suite -o scannertest/scanner_spec.gen.go Scanner

// Item is a scanned value.
type Item struct {
	ID   string
	Data []byte
}

// Scanner exercises iterator return types.
type Scanner interface {
	// Keys returns all keys as a pull iterator.
	Keys(ctx context.Context) iter.Seq[string]

	// Scan returns items with potential error.
	Scan(ctx context.Context, prefix string) iter.Seq2[Item, error]

	//testkit:ctx
	// Count returns the number of items.
	Count(ctx context.Context) (int, error)
}
