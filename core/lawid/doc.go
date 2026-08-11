// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package lawid holds the identifiers every auto-derived law reports under.
//
// One home for a vocabulary two modules spell. [engine/model/law] returns these
// from `ID()`, so a failing run names one; the model generator selects laws by
// them, so a generated binding names the same one. Before this package the two
// were independent string literals in modules that do not depend on each other,
// and a law renamed on one side left a stale identifier on the other that no
// compiler and no test could see.
//
// # Why the root module
//
// `engine` and `generator` both depend on `go.thesmos.sh/testkit` and neither
// depends on the other (docs/adr/0005). This package is the only placement that
// both can import, and it costs nothing to import: string constants, no
// dependencies, no initialisation.
//
// # What is not here
//
// `REQ-*` tags. [engine/model/law.Law]'s `REQID` is empty for every law in this
// package and carries a requirement identifier only where a consumer tagged
// one. Those name requirements in the consumer's own tracker, so testkit cannot
// enumerate them and does not try.
//
// # Stability
//
// An identifier appears in test output, in a skipped subtest's name, and in the
// `<Iface>Without("model/<id>")` a consumer writes to drop one law. That makes
// it consumer-facing text: renaming a constant is free, and changing its
// *value* breaks a consumer's drop list silently. Treat the strings as the
// published surface and the constant names as the private one.
package lawid
