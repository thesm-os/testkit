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

	fixture := publisheratmostoncetest.DefaultContractFixture()

	publisheratmostoncetest.AssertContractContract(t,
		publisheratmostoncetest.ContractModel(),
		publisheratmostoncetest.ContractSubject("in-memory", func() publisheratmostonce.Contract {
			return publisheratmostoncetest.NewInMemory()
		}),
		publisheratmostoncetest.ContractOnSubscribe("receives what is published after it attaches", func(
			tb testing.TB, subject publisheratmostonce.Contract,
		) {
			tb.Helper()
			stream, err := subject.Subscribe(tb.Context())
			testkit.NoError(tb, err, "a subscriber attaches")

			testkit.NoError(tb, subject.Publish(tb.Context(), fixture.VOther),
				"a message published afterwards is accepted")
			testkit.Equal(tb, <-stream, fixture.VOther, "and reaches them")
		}),
		publisheratmostoncetest.ContractOnPublish("reports a subscriber it can no longer reach", func(
			tb testing.TB, subject publisheratmostonce.Contract, v publisheratmostonce.Value,
		) {
			tb.Helper()
			_, err := subject.Subscribe(tb.Context())
			testkit.NoError(tb, err, "a subscriber attaches")

			for range 32 {
				if err := subject.Publish(tb.Context(), v); err != nil {
					testkit.ErrorIs(tb, err, publisheratmostoncetest.ErrFull,
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

	publisheratmostoncetest.AssertContractContract(t,
		publisheratmostoncetest.ContractSubject("in-memory", func() publisheratmostonce.Contract {
			return publisheratmostoncetest.NewInMemory()
		}),
		publisheratmostoncetest.ContractWithout("Publish/smoke"),
		publisheratmostoncetest.ContractWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	publisheratmostoncetest.ContractModelSaturation(t, func() publisheratmostonce.Contract {
		return publisheratmostoncetest.NewInMemory()
	})
}
