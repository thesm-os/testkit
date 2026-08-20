// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package publisherexactlyoncetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	publisherexactlyonce "go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-exactlyonce"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-exactlyonce/publisherexactlyoncetest"
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

	publisherexactlyoncetest.RunContract(
		t,
		publisherexactlyoncetest.ContractHarness[*publisherexactlyoncetest.InMemory]{
			Name: "in-memory",
			New:  publisherexactlyoncetest.NewInMemory,
		},
		publisherexactlyoncetest.ContractChecks{
			{
				Method: "Replay",
				Name:   "suppresses-the-duplicate",
				Claim:  "Replay suppresses the duplicate and repairs the loss",
				Run: func(tb testing.TB, s publisherexactlyonce.Contract, fx publisherexactlyoncetest.ContractFixture) {
					tb.Helper()
					stream, err := s.Subscribe(tb.Context())
					testkit.NoError(tb, err, "a subscriber attaches")

					testkit.NoError(tb, s.Publish(tb.Context(), fx.Value()), "the original lands")
					testkit.NoError(tb, s.Replay(tb.Context(), fx.Value()), "a replay of it is accepted")
					testkit.NoError(tb, s.Replay(tb.Context(), fx.ValueOther()),
						"and so is a replay of a message that was lost")

					testkit.Equal(tb, <-stream, fx.Value(), "the subscriber takes the original once")
					testkit.Equal(tb, <-stream, fx.ValueOther(),
						"then the repaired loss — never the duplicate")
				},
			},
			{
				Method: "Subscribe",
				Name:   "receives-what-is-published-after",
				Claim:  "Subscribe receives what is published after it attaches",
				Run: func(tb testing.TB, s publisherexactlyonce.Contract, fx publisherexactlyoncetest.ContractFixture) {
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
				Run: func(tb testing.TB, s publisherexactlyonce.Contract, fx publisherexactlyoncetest.ContractFixture) {
					tb.Helper()
					_, err := s.Subscribe(tb.Context())
					testkit.NoError(tb, err, "a subscriber attaches")

					for range 32 {
						if err := s.Publish(tb.Context(), fx.Value()); err != nil {
							testkit.ErrorIs(tb, err, publisherexactlyoncetest.ErrFull,
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

	publisherexactlyoncetest.RunContract(
		t,
		publisherexactlyoncetest.ContractHarness[*publisherexactlyoncetest.InMemory]{
			Name: "in-memory",
			New:  publisherexactlyoncetest.NewInMemory,
		},
		publisherexactlyoncetest.ContractSuite.Without(publisherexactlyoncetest.ContractSuite.Checks.Publish.Smoke()),
	)
}
