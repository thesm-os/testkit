// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/rand"
)

// Configurable is the part of a per-method stub that does not depend on the
// method's signature.
//
// A generated double holds one [MethodStub] per method, each with a different
// type parameter, so it cannot loop over them as a homogeneous slice. Every
// setting that applies to the whole double — strict mode, a virtual clock, a
// seeded random source, bench mode — then has to be written once per method,
// and a thirteen-method interface pays that four times over.
//
// This interface is what makes the loop possible. [MethodStub] satisfies it
// for any call type, so a double can keep `[]Configurable` alongside its
// typed fields and apply a setting across all of them in three lines rather
// than N.
//
// It carries only the methods that return nothing. The chaining forms —
// [MethodStub.WithClock] and [MethodStub.WithRandSource] — return the stub so
// a caller can configure one method fluently, which is the right shape for
// that use and the wrong one for an interface.
type Configurable interface {
	// Strict makes an unconfigured call fail the test rather than return a
	// zero value.
	Strict()

	// Reset clears recorded calls, fault counters, and call-count
	// expectations, leaving Func and Returns configuration in place.
	Reset()

	// Verify checks the call-count expectations. A generated constructor
	// registers this as a cleanup so an unmet Times is reported without the
	// caller remembering to ask.
	Verify()

	// BenchMode disables call recording, whose allocations are what a
	// benchmark would otherwise be measuring.
	BenchMode()

	// SetClock drives latency and time-windowed faults from clk.
	SetClock(clk clock.Clock)

	// SetRandSource makes probabilistic fault injection reproducible.
	SetRandSource(src rand.Source)
}

// SetClock is the non-chaining form of [MethodStub.WithClock], for applying a
// clock across every method of a double through [Configurable].
func (s *MethodStub[C]) SetClock(clk clock.Clock) { s.WithClock(clk) }

// SetRandSource is the non-chaining form of [MethodStub.WithRandSource], for
// applying a source across every method of a double through [Configurable].
func (s *MethodStub[C]) SetRandSource(src rand.Source) { s.WithRandSource(src) }

// Compile-time confirmation that a method stub is configurable without
// knowing its call type — the property the whole interface exists for.
var _ Configurable = (*MethodStub[struct{}])(nil)
