// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"testing"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/rand"
	"go.thesmos.sh/testkit/stub"
)

// fanoutMember records which group-wide settings reached it.
type fanoutMember struct {
	strict, reset, verified, bench bool
	clocked, randed                bool
}

func (m *fanoutMember) Strict()                   { m.strict = true }
func (m *fanoutMember) Reset()                    { m.reset = true }
func (m *fanoutMember) Verify()                   { m.verified = true }
func (m *fanoutMember) BenchMode()                { m.bench = true }
func (m *fanoutMember) SetClock(clock.Clock)      { m.clocked = true }
func (m *fanoutMember) SetRandSource(rand.Source) { m.randed = true }

// TestGroupFansOut pins the one job: every setting reaches every member.
func TestGroupFansOut(t *testing.T) {
	t.Parallel()

	a, b := &fanoutMember{}, &fanoutMember{}
	g := stub.NewGroup(a, b)
	g.StrictAll()
	g.Reset()
	g.BenchMode()
	g.SetClock(clock.RealClock())
	g.SetRandSource(rand.DefaultRandSource())

	for i, m := range []*fanoutMember{a, b} {
		if !m.strict || !m.reset || !m.bench || !m.clocked || !m.randed {
			t.Errorf("member %d missed a fan-out: %+v", i, m)
		}
	}
}

// TestGroupBindVerifiesAtCleanup pins the constructor tail every double
// used to restate: Bind's cleanup verifies each member when the test
// ends, and a nil tb skips the registration entirely.
//
//nolint:tparallel // the subtest must complete, cleanups included, before the assertion reads the flag; a parallel subtest defers past the read.
func TestGroupBindVerifiesAtCleanup(t *testing.T) {
	t.Parallel()

	m := &fanoutMember{}
	t.Run("bound", func(t *testing.T) {
		stub.NewGroup(m).Bind(t)
	})
	if !m.verified {
		t.Error("Bind must verify call-count expectations when the test ends")
	}

	stub.NewGroup(&fanoutMember{}).Bind(nil) // must not panic: benchmarks pass nil
}
