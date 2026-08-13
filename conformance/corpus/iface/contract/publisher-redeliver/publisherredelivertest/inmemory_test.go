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

	publisherredelivertest.AssertContractContract(t,
		publisherredelivertest.ContractModel(),
		publisherredelivertest.ContractSubject("in-memory", func() publisherredeliver.Contract {
			return publisherredelivertest.NewInMemory()
		}),
		publisherredelivertest.ContractOnRepublish("duplicates what was already delivered", func(
			tb testing.TB, subject publisherredeliver.Contract, v publisherredeliver.Value,
		) {
			tb.Helper()
			stream, err := subject.Subscribe(tb.Context())
			testkit.NoError(tb, err, "a subscriber attaches")

			testkit.NoError(tb, subject.Publish(tb.Context(), v), "the original lands")
			testkit.NoError(tb, subject.Republish(tb.Context(), v), "and so does the redelivery")
			testkit.Equal(tb, <-stream, v, "the subscriber takes the original")
			testkit.Equal(tb, <-stream, v, "and the duplicate at-least-once permits")
		}),
		publisherredelivertest.ContractOnPublish("reports a subscriber it can no longer reach", func(
			tb testing.TB, subject publisherredeliver.Contract, v publisherredeliver.Value,
		) {
			tb.Helper()
			_, err := subject.Subscribe(tb.Context())
			testkit.NoError(tb, err, "a subscriber attaches")

			for range 32 {
				if err := subject.Publish(tb.Context(), v); err != nil {
					testkit.ErrorIs(tb, err, publisherredelivertest.ErrFull,
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

	publisherredelivertest.AssertContractContract(t,
		publisherredelivertest.ContractSubject("in-memory", func() publisherredeliver.Contract {
			return publisherredelivertest.NewInMemory()
		}),
		publisherredelivertest.ContractWithout("Publish/smoke"),
		publisherredelivertest.ContractWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	publisherredelivertest.ContractModelSaturation(t, func() publisherredeliver.Contract {
		return publisherredelivertest.NewInMemory()
	})
}
