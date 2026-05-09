// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ifacevalue exercises the suite generator's type rendering
// against an interface returning an interface-typed value
// (io.Reader). Stresses interface-typed value rendering — the
// sample defaulter falls back to nil for interface types, so the
// contract's "returns for key" baseline asserts equality on a nil
// io.Reader (which compares equal under reflect.DeepEqual).
package ifacevalue

//go:generate testkit suite -o ifacevaluetest/cache_spec.gen_test.go ReaderCache

import (
	"context"
	"io"
)

// ReaderCache is a key→io.Reader cache.
type ReaderCache interface {
	// Reader-shape with V = io.Reader.
	Get(ctx context.Context, key string) (io.Reader, error)
}
