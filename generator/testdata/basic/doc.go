// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package basic is the canonical multi-purpose fixture used by every
// generator's tests. Files are split by concern:
//
//   - errors.go   — sentinel error vars, custom error types
//   - store.go    — Store interface + Item value type
//   - counter.go  — Counter struct + methods
//   - status.go   — Status enum
//
// Generators that don't apply to some content (e.g. sentinel against
// the Store interface) simply ignore it. The fixture's purpose is to
// be a single source of truth for "what does the world look like" so
// every generator runs against the same realistic surface.
//
//testkit:sentinel-no-overlap-with go.thesmos.sh/testkit/generator/testdata/storage
package basic
