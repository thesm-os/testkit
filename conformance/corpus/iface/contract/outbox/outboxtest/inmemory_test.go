// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package outboxtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/outbox"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/outbox/outboxtest"
)

// outbox is the suite tier's under ADR-0018. `AUTO-PUBLISHER-DELIVERS` states
// delivery to a subscriber that was already listening, which is the `publisher`
// contract; what an outbox adds is that the record survives until somebody is,
// and no law carries that.
//
// So the generated check appends first and subscribes second. It appends twice,
// because the harness seeds every fresh subject through Append — a subject that
// took this check's own record and dropped it would still deliver the seed's
// and pass.
func TestContractContract(t *testing.T) {
	t.Parallel()

	fixture := outboxtest.DefaultContractFixture()

	outboxtest.AssertContractContract(t,
		outboxtest.ContractModel(),
		outboxtest.ContractSubject("in-memory", func() outbox.Contract {
			return outboxtest.NewInMemory()
		}),
		outboxtest.ContractOnSubscribe("delivers to a subscriber that was already attached", func(
			tb testing.TB, subject outbox.Contract,
		) {
			tb.Helper()
			// The live half, which the generated check deliberately does not
			// state: it attaches last so that the durability claim is the one
			// being made. An outbox owes both.
			stream, err := subject.Subscribe(tb.Context())
			testkit.NoError(tb, err, "a subscriber attaches")

			// The seed's record is already waiting, so drain it before the one
			// this check is about.
			<-stream

			testkit.NoError(tb, subject.Append(tb.Context(), fixture.VOther),
				"a record appended after the subscriber attached is accepted")
			testkit.Equal(tb, <-stream, fixture.VOther, "and reaches them")
		}),
		outboxtest.ContractOnAppend("reports a subscriber it can no longer reach", func(
			tb testing.TB, subject outbox.Contract, v outbox.Value,
		) {
			tb.Helper()
			// A silent drop is the one failure this whole fixture is about, so
			// the drop has to be reported — and a report nothing reaches is a
			// report nothing proves. The reader never reads, which is what puts
			// the subscriber far enough behind.
			_, err := subject.Subscribe(tb.Context())
			testkit.NoError(tb, err, "a subscriber attaches")

			for range 32 {
				if err := subject.Append(tb.Context(), v); err != nil {
					testkit.ErrorIs(tb, err, outboxtest.ErrFull,
						"the append says why it could not be taken")
					return
				}
			}
			tb.Fatalf("a subscriber that never reads was never reported as behind")
		}),
		outboxtest.ContractOnSubscribe("refuses a subscriber it cannot hand the backlog to", func(
			tb testing.TB, subject outbox.Contract,
		) {
			tb.Helper()
			// The other half: the backlog is loaded at subscribe time, so a log
			// longer than a subscriber can hold is refused rather than
			// truncated — which would lose exactly the records an outbox exists
			// to keep.
			for range 32 {
				if err := subject.Append(tb.Context(), outbox.Value{Key: "k", Body: "b"}); err != nil {
					tb.Fatalf("appending with nobody listening must not fail: %v", err)
				}
			}
			_, err := subject.Subscribe(tb.Context())
			testkit.ErrorIs(tb, err, outboxtest.ErrFull,
				"a backlog too long to hand over is refused rather than truncated")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	outboxtest.AssertContractContract(t,
		outboxtest.ContractSubject("in-memory", func() outbox.Contract {
			return outboxtest.NewInMemory()
		}),
		outboxtest.ContractWithout("Append/smoke"),
		outboxtest.ContractWithoutDouble(),
	)
}
