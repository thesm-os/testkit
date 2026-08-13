// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package publisheratleastoncetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	publisheratleastonce "go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-atleastonce"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-atleastonce/publisheratleastoncetest"
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

	fixture := publisheratleastoncetest.DefaultContractFixture()

	publisheratleastoncetest.AssertContractContract(t,
		publisheratleastoncetest.ContractModel(),
		publisheratleastoncetest.ContractSubject("in-memory", func() publisheratleastonce.Contract {
			return publisheratleastoncetest.NewInMemory()
		}),
		publisheratleastoncetest.ContractOnSubscribe("receives what is published after it attaches", func(
			tb testing.TB, subject publisheratleastonce.Contract,
		) {
			tb.Helper()
			stream, err := subject.Subscribe(tb.Context())
			testkit.NoError(tb, err, "a subscriber attaches")

			testkit.NoError(tb, subject.Publish(tb.Context(), fixture.VOther),
				"a message published afterwards is accepted")
			testkit.Equal(tb, <-stream, fixture.VOther, "and reaches them")
		}),
		publisheratleastoncetest.ContractOnPublish("reports a subscriber it can no longer reach", func(
			tb testing.TB, subject publisheratleastonce.Contract, v publisheratleastonce.Value,
		) {
			tb.Helper()
			_, err := subject.Subscribe(tb.Context())
			testkit.NoError(tb, err, "a subscriber attaches")

			for range 32 {
				if err := subject.Publish(tb.Context(), v); err != nil {
					testkit.ErrorIs(tb, err, publisheratleastoncetest.ErrFull,
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

	publisheratleastoncetest.AssertContractContract(t,
		publisheratleastoncetest.ContractSubject("in-memory", func() publisheratleastonce.Contract {
			return publisheratleastoncetest.NewInMemory()
		}),
		publisheratleastoncetest.ContractWithout("Publish/smoke"),
		publisheratleastoncetest.ContractWithoutDouble(),
	)
}
