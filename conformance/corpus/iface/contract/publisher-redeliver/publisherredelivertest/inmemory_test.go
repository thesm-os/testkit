// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package publisherredelivertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	publisherredeliver "go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-redeliver"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-redeliver/publisherredelivertest"
)

// publisher-redeliver is the model tier's under ADR-0018, and the one member
// of the mode family whose `AUTO-PUBLISHER-AT-LEAST-ONCE` runs its redelivery
// arm: the law re-offers the published message through Republish and counts
// the duplicate the mode permits. Its siblings prove the role's omission.
func TestContractContract(t *testing.T) {
	t.Parallel()

	publisherredelivertest.RunContract(
		t,
		publisherredelivertest.ContractHarness[*publisherredelivertest.InMemory]{
			Name: "in-memory",
			New:  publisherredelivertest.NewInMemory,
		},
		publisherredelivertest.ContractChecks{
			{
				Method: "Republish",
				Name:   "duplicates-what-was-delivered",
				Claim:  "Republish duplicates what was already delivered",
				Run: func(tb testing.TB, s publisherredeliver.Contract, fx publisherredelivertest.ContractFixture) {
					tb.Helper()
					stream, err := s.Subscribe(tb.Context())
					testkit.NoError(tb, err, "a subscriber attaches")

					testkit.NoError(tb, s.Publish(tb.Context(), fx.Value()), "the original lands")
					testkit.NoError(tb, s.Republish(tb.Context(), fx.Value()), "and so does the redelivery")
					testkit.Equal(tb, <-stream, fx.Value(), "the subscriber takes the original")
					testkit.Equal(tb, <-stream, fx.Value(), "and the duplicate at-least-once permits")
				},
			},
			{
				Method: "Subscribe",
				Name:   "receives-what-is-published-after",
				Claim:  "Subscribe receives what is published after it attaches",
				Run: func(tb testing.TB, s publisherredeliver.Contract, fx publisherredelivertest.ContractFixture) {
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
				Run: func(tb testing.TB, s publisherredeliver.Contract, fx publisherredelivertest.ContractFixture) {
					tb.Helper()
					_, err := s.Subscribe(tb.Context())
					testkit.NoError(tb, err, "a subscriber attaches")

					for range 32 {
						if err := s.Publish(tb.Context(), fx.Value()); err != nil {
							testkit.ErrorIs(tb, err, publisherredelivertest.ErrFull,
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

	publisherredelivertest.RunContract(
		t,
		publisherredelivertest.ContractHarness[*publisherredelivertest.InMemory]{
			Name: "in-memory",
			New:  publisherredelivertest.NewInMemory,
		},
		publisherredelivertest.ContractSuite.Without(publisherredelivertest.ContractSuite.Checks.Publish.Smoke()),
	)
}
