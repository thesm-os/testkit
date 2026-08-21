// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package stamp is the read side of testkit's annotator directives: the
// metadata keys `//testkit:default` and `//testkit:role` write, and the
// one function per key that reads them back.
//
// Separate from the annotators that write them, because writing and
// reading a stamp have different audiences. The annotator is a pipeline
// stage — it owns the directive schema, the parse, the diagnostics, and
// a version that is a cache key. A generator reading the stamp wants two
// strings. Importing the annotator to get them drags in the plugin
// constructor as well, which is how a generator ends up in a position to
// register a second copy of an annotator it only meant to read.
//
// One function per key, and no exported way to reach the key itself
// through this package's callers. A reader that goes to [sdk.Key.Get]
// directly gets the presence flag alongside the value and has to decide
// what an absent key means, which is a decision the stamp's owner has
// already made — for both of these, the empty string is the absence, and
// a caller that re-decides it can differ.
package stamp
