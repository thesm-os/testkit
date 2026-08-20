// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package leaderelectiontest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	leaderelection "go.thesmos.sh/testkit/conformance/corpus/iface/contract/leader-election"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/leader-election/leaderelectiontest"
)

// leader-election is owned by no tier under ADR-0018, which the gate reports as
// a law to write rather than a check to invent.
//
// "Exactly one leader" is a property of a group, and a check receives one
// subject. A generated check would campaign uncontested and report success —
// which every implementation manages, including one that never looks at whether
// anybody else holds it. The losing side is reached by declaring a second
// subject whose registry somebody already leads.
func TestContractContract(t *testing.T) {
	t.Parallel()

	leaderelectiontest.RunContract(t,
		leaderelectiontest.ContractHarness[*leaderelectiontest.InMemory]{
			Name: "in-memory", New: leaderelectiontest.NewInMemory,
		},
		leaderelectiontest.ContractHarness[*leaderelectiontest.InMemory]{
			Name: "in-memory, contended",
			// A constructor may build any starting state, which is how a
			// group property is reached from a surface that hands out one
			// subject.
			New: func() *leaderelectiontest.InMemory {
				r := leaderelectiontest.NewRegistry()
				incumbent := r.Candidate()
				if err := incumbent.Campaign(t.Context()); err != nil {
					panic("leaderelectiontest_test: seating the incumbent: " + err.Error())
				}
				return r.Candidate()
			},
		},
		leaderelectiontest.ContractChecks{
			{
				Method: "IsLeader",
				Name:   "answers-a-caller-who-gave-up",
				Claim:  "IsLeader answers a caller who already gave up",
				Run: func(tb testing.TB, s leaderelection.Contract, fx leaderelectiontest.ContractFixture) {
					tb.Helper()
					// IsLeader returns no error, so the generated family asks
					// only that it survive a nil context — a cancelled one it
					// never passes. A leader that answered "yes" to a caller who
					// had already stopped waiting would have them act on a claim
					// nobody is holding.
					ctx, cancel := context.WithCancel(tb.Context())
					cancel()
					testkit.False(tb, s.IsLeader(ctx),
						"a cancelled caller is told nothing rather than told yes")
				},
			},
			{
				Method: "Campaign",
				Name:   "loses-to-a-standing-leader",
				Claim:  "Campaign loses to a standing leader",
				Run: func(tb testing.TB, s leaderelection.Contract, fx leaderelectiontest.ContractFixture) {
					tb.Helper()
					// True of the contended subject and not of the lone one,
					// which is why both are declared: a check that held for
					// either alone would state half the contract.
					if err := s.Campaign(tb.Context()); err != nil {
						testkit.ErrorIs(tb, err, leaderelectiontest.ErrHeld,
							"a campaign that loses says who to")
						testkit.False(tb, s.IsLeader(tb.Context()), "and does not claim otherwise")
						testkit.NoError(tb, s.Resign(tb.Context()),
							"while standing down from something never held is not a failure")
						return
					}
					testkit.True(tb, s.IsLeader(tb.Context()), "a campaign that wins takes the leadership")
					testkit.NoError(tb, s.Resign(tb.Context()), "and can stand down")
					testkit.False(tb, s.IsLeader(tb.Context()), "which gives it up")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	leaderelectiontest.RunContract(t,
		leaderelectiontest.ContractHarness[*leaderelectiontest.InMemory]{
			Name: "in-memory", New: leaderelectiontest.NewInMemory,
		},
		leaderelectiontest.ContractSuite.Without(leaderelectiontest.ContractSuite.Checks.Campaign.Smoke()),
	)
}
