// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package nestedmap exercises the suite generator's type rendering
// against an interface whose value type is a nested map
// (map[string]int). Stresses map literal sample rendering and
// reflect.DeepEqual on map values.
package nestedmap

//go:generate testkit suite -o nestedmaptest/cache_spec.gen_test.go MapCache

import "context"

// MapCache is a key→map[string]int cache.
type MapCache interface {
	// Reader-shape with V = map[string]int.
	Get(ctx context.Context, key string) (map[string]int, error)
}
