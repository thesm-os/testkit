// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

// Tag is a string-typed enum. Used by enum-generator tests to
// confirm analyze rejects non-integer underlying types up front
// (wire-compat asserts an integer mapping; emitting `int(<Tag>)`
// would fail to compile). Deliberately has no `go:generate`
// directive — the generator must error on this type, not produce
// output.
type Tag string

const (
	TagAlpha Tag = "alpha"
	TagBeta  Tag = "beta"
)
