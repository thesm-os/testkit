// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package seededreader is the language-axis fixture for the SEED SEAM:
// an interface nothing can write to.
//
// A run populates a subject through the surface under test, and this
// surface offers no way in. A read against an empty subject cannot tell
// a miss from a bug, so the harness receives the corpus instead of a
// bare constructor — the suite decides what is seeded, from its own
// pools, and the harness decides how loading happens.
//
// Both roles are stamped because the corpus is zipped from them: a key
// pool with no payload pool names half a map.
package seededreader

import "context"

// Key identifies one document.
//
//testkit:role key
//testkit:default "test-doc"
type Key string

// Body is what a document holds.
//
//testkit:role payload
//testkit:default "test-contents"
type Body string

// Catalog reads documents and writes none.
//
//testkit:out seededreadertest/ pkg=seededreadertest
//testkit:stub
//testkit:suite
type Catalog interface {
	// Lookup answers for a seeded key and reports a miss for any other.
	Lookup(ctx context.Context, key Key) (Body, error)

	// Len reports how many documents were loaded.
	Len(ctx context.Context) (int, error)
}
