// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package rand provides deterministic random number sources for tests.
//
// [RandSource] is the interface consumed by probabilistic fault strategies.
// [DefaultRandSource] uses the standard library. [FixedRandSource] returns
// a fixed value for deterministic testing.
package rand
