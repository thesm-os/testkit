// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package nestedgeneric exercises the suite generator's type
// rendering against an interface returning a generic type
// instantiated with another generic type (Page[Item]). Stresses
// nested generic value rendering — type-arg propagation through
// the Reader baseline emission.
package nestedgeneric

//go:generate testkit suite -o nestedgenerictest/cache_spec.gen_test.go PageCache

import "context"

// Item is the value type carried by a Page.
type Item struct {
	ID string
}

// Page is a generic page-of-T carrier. The cache returns
// Page[Item] — testing the generator's nested generic rendering.
type Page[T any] struct {
	Items  []T
	Cursor string
}

// PageCache is a key→Page[Item] cache.
type PageCache interface {
	// Reader-shape with V = Page[Item].
	Get(ctx context.Context, key string) (Page[Item], error)
}
