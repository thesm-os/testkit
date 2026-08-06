// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/rand"
	"go.thesmos.sh/testkit/stub"
)

// A generated double applies whole-double settings by looping over this
// interface rather than naming every method, so the non-chaining setters have
// to reach the same state the fluent ones do.
func TestConfigurable(t *testing.T) {
	t.Parallel()

	t.Run("sets the clock across the interface", func(t *testing.T) {
		t.Parallel()
		var c stub.Configurable = stub.NewMethodStub[call](t, "Fake.Get")
		clk := clock.NewTestClock(time.Unix(0, 0))

		c.SetClock(clk)

		testkit.Equal(t, c.(*stub.MethodStub[call]).Clock(), clock.Clock(clk),
			"SetClock must reach the same state as WithClock")
	})

	t.Run("sets the random source across the interface", func(t *testing.T) {
		t.Parallel()
		// Probabilistic faults are the only consumer, and a source that never
		// landed leaves them unreproducible rather than visibly broken.
		var c stub.Configurable = stub.NewMethodStub[call](t, "Fake.Get")

		c.SetRandSource(rand.FixedRandSource(0.5))

		s := c.(*stub.MethodStub[call])
		s.FaultsWithProbability(1.0, testkit.TestError("configurable"))
		fired, _ := s.ShouldFaultFor(call{})
		testkit.True(t, fired, "a certain fault must fire once a source is set")
	})
}
