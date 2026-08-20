// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package publisheratmostoncetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	publisheratmostonce "go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-atmostonce"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-atmostonce/publisheratmostoncetest"
)

// publisher is the model tier's under ADR-0018: `AUTO-PUBLISHER-DELIVERS` and
// the three delivery-guarantee laws state it.
//
// Delivery is what this contract shares with `outbox`, and the difference is
// what neither signature shows: an outbox holds a record until somebody reads
// it, a publisher delivers to whoever is listening. The suite tier owns the
// outbox half because no law states durability; this half is already covered.
func TestContractContract(t *testing.T) {
	t.Parallel()

	publisheratmostoncetest.RunContract(
		t,
		publisheratmostoncetest.ContractHarness[*publisheratmostoncetest.InMemory]{
			Name: "in-memory",
			New:  publisheratmostoncetest.NewInMemory,
		},
		publisheratmostoncetest.ContractChecks{
			{
				Method: "Subscribe",
				Name:   "receives-what-is-published-after",
				Claim:  "Subscribe receives what is published after it attaches",
				Run: func(tb testing.TB, s publisheratmostonce.Contract, fx publisheratmostoncetest.ContractFixture) {
					tb.Helper()
					stream, err := s.Subscribe(tb.Context())
					testkit.NoError(tb, err, "a subscriber attaches")

					testkit.NoError(tb, s.Publish(tb.Context(), fx.ValueOther()),
						"a message published afterwards is accepted")
					testkit.Equal(tb, <-stream, fx.ValueOther(), "and reaches them")
				},
			},
			{
				Method: "Publish",
				Name:   "reports-an-unreachable-subscriber",
				Claim:  "Publish reports a subscriber it can no longer reach",
				Run: func(tb testing.TB, s publisheratmostonce.Contract, fx publisheratmostoncetest.ContractFixture) {
					tb.Helper()
					_, err := s.Subscribe(tb.Context())
					testkit.NoError(tb, err, "a subscriber attaches")

					for range 32 {
						if err := s.Publish(tb.Context(), fx.Value()); err != nil {
							testkit.ErrorIs(tb, err, publisheratmostoncetest.ErrFull,
								"the publish says why it could not be taken")
							return
						}
					}
					tb.Fatalf("a subscriber that never reads was never reported as behind")
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

	publisheratmostoncetest.RunContract(
		t,
		publisheratmostoncetest.ContractHarness[*publisheratmostoncetest.InMemory]{
			Name: "in-memory",
			New:  publisheratmostoncetest.NewInMemory,
		},
		publisheratmostoncetest.ContractSuite.Without(publisheratmostoncetest.ContractSuite.Checks.Publish.Smoke()),
	)
}
