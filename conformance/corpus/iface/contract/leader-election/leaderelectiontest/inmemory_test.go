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
// "Exactly one leader" is a property of a group, and the harness builds one
// subject from one factory. A generated check would campaign uncontested and
// report success — which every implementation manages, including one that never
// looks at whether anybody else holds it.
func TestContractContract(t *testing.T) {
	t.Parallel()

	leaderelectiontest.AssertContractContract(t,
		leaderelectiontest.ContractModel(),
		leaderelectiontest.ContractSubject("in-memory", func() leaderelection.Contract {
			return leaderelectiontest.NewInMemory()
		}),
		leaderelectiontest.ContractSubject("in-memory, contended", func() leaderelection.Contract {
			// Contention is a property of the group, and a check receives one
			// subject — but a factory may build any starting state, so the
			// losing side of an election is reached by handing the run a
			// candidate whose registry somebody else already leads.
			//
			// This is what the extension point is for, and what the model
			// generator will drive when it lands: the subject keeps the
			// behaviour, and the suite reaches it through a second subject
			// rather than through a second call.
			r := leaderelectiontest.NewRegistry()
			incumbent := r.Candidate()
			if err := incumbent.Campaign(t.Context()); err != nil {
				panic("leaderelectiontest_test: seating the incumbent: " + err.Error())
			}
			return r.Candidate()
		}),
		leaderelectiontest.ContractOnIsLeader("answers a caller who already gave up", func(
			tb testing.TB, subject leaderelection.Contract,
		) {
			tb.Helper()
			// IsLeader returns no error, so the generated family asks only that
			// it survive a nil context — a cancelled one it never passes. A
			// leader that answered "yes" to a caller who had already stopped
			// waiting would have them act on a claim nobody is holding.
			ctx, cancel := context.WithCancel(tb.Context())
			cancel()
			testkit.False(tb, subject.IsLeader(ctx),
				"a cancelled caller is told nothing rather than told yes")
		}),
		leaderelectiontest.ContractOnCampaign("loses to a standing leader", func(
			tb testing.TB, subject leaderelection.Contract,
		) {
			tb.Helper()
			// True of the contended subject and not of the lone one, which is
			// why both are declared: a check that held for either alone would
			// state half the contract.
			if err := subject.Campaign(tb.Context()); err != nil {
				testkit.ErrorIs(tb, err, leaderelectiontest.ErrHeld,
					"a campaign that loses says who to")
				testkit.False(tb, subject.IsLeader(tb.Context()), "and does not claim otherwise")
				testkit.NoError(tb, subject.Resign(tb.Context()),
					"while standing down from something never held is not a failure")
				return
			}
			testkit.True(tb, subject.IsLeader(tb.Context()), "a campaign that wins takes the leadership")
			testkit.NoError(tb, subject.Resign(tb.Context()), "and can stand down")
			testkit.False(tb, subject.IsLeader(tb.Context()), "which gives it up")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	leaderelectiontest.AssertContractContract(t,
		leaderelectiontest.ContractSubject("in-memory", func() leaderelection.Contract {
			return leaderelectiontest.NewInMemory()
		}),
		leaderelectiontest.ContractWithout("Campaign/smoke"),
		leaderelectiontest.ContractWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	leaderelectiontest.ContractModelSaturation(t, func() leaderelection.Contract {
		return leaderelectiontest.NewInMemory()
	})
}
