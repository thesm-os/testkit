// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package publishertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher/publishertest"
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

	fixture := publishertest.DefaultContractFixture()

	publishertest.AssertContractContract(t,
		publishertest.ContractModel(),
		publishertest.ContractSubject("in-memory", func() publisher.Contract {
			return publishertest.NewInMemory()
		}),
		publishertest.ContractOnSubscribe("receives what is published after it attaches", func(
			tb testing.TB, subject publisher.Contract,
		) {
			tb.Helper()
			stream, err := subject.Subscribe(tb.Context())
			testkit.NoError(tb, err, "a subscriber attaches")

			testkit.NoError(tb, subject.Publish(tb.Context(), fixture.VOther),
				"a message published afterwards is accepted")
			testkit.Equal(tb, <-stream, fixture.VOther, "and reaches them")
		}),
		publishertest.ContractOnPublish("reports a subscriber it can no longer reach", func(
			tb testing.TB, subject publisher.Contract, v publisher.Value,
		) {
			tb.Helper()
			_, err := subject.Subscribe(tb.Context())
			testkit.NoError(tb, err, "a subscriber attaches")

			for range 32 {
				if err := subject.Publish(tb.Context(), v); err != nil {
					testkit.ErrorIs(tb, err, publishertest.ErrFull,
						"the publish says why it could not be taken")
					return
				}
			}
			tb.Fatalf("a subscriber that never reads was never reported as behind")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	publishertest.AssertContractContract(t,
		publishertest.ContractSubject("in-memory", func() publisher.Contract {
			return publishertest.NewInMemory()
		}),
		publishertest.ContractWithout("Publish/smoke"),
		publishertest.ContractWithoutDouble(),
	)
}
