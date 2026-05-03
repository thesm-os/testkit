// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package rand

import "math/rand/v2"

// Source abstracts random number generation for probabilistic fault
// injection. testkit defines the interface; consumers inject their own
// deterministic RNG (e.g., a simulation engine's seeded PCG) via
// [MethodStub.WithRandSource].
//
// Implementations must be safe for concurrent use. testkit calls Float64
// from generated stub dispatch code which may run concurrently across
// goroutines. [DefaultRandSource] (math/rand/v2 global) is thread-safe.
//
// The consumer's RandSource implementation is responsible for seed
// lifecycle — testkit does not manage seeding.
type Source interface {
	// Float64 returns a pseudo-random float64 in the half-open interval [0.0, 1.0).
	Float64() float64
}

// defaultRandSource wraps math/rand/v2's global generator.
type defaultRandSource struct{}

// DefaultRandSource returns a [RandSource] backed by [math/rand/v2].
func DefaultRandSource() Source {
	return defaultRandSource{}
}

func (defaultRandSource) Float64() float64 { return rand.Float64() } //nolint:gosec // fault injection does not need crypto-grade randomness

// FixedRandSource returns a [RandSource] that always returns the given value.
// Useful for deterministic testing of probabilistic fault strategies.
func FixedRandSource(v float64) Source {
	return fixedRandSource{v: v}
}

type fixedRandSource struct{ v float64 }

func (f fixedRandSource) Float64() float64 { return f.v }
