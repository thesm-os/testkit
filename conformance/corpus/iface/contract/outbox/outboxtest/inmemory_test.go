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
// So the durability row appends first and subscribes second, and the live row
// does the reverse. An outbox owes both.
func TestContractContract(t *testing.T) {
	t.Parallel()

	outboxtest.RunContract(t,
		outboxtest.ContractHarness[*outboxtest.InMemory]{Name: "in-memory", New: outboxtest.NewInMemory},
		outboxtest.ContractChecks{
			{
				Method: "Subscribe",
				Name:   "keeps-a-record-until-somebody-listens",
				Claim:  "Subscribe hands over a record appended before it attached",
				Run: func(tb testing.TB, s outbox.Contract, fx outboxtest.ContractFixture) {
					tb.Helper()
					// The durability half: the record is written with nobody
					// listening, which is the whole reason an outbox exists.
					testkit.NoError(tb, s.Append(tb.Context(), fx.Value()),
						"a record is appended with nobody listening")

					stream, err := s.Subscribe(tb.Context())
					testkit.NoError(tb, err, "a subscriber attaches afterwards")
					testkit.Equal(tb, <-stream, fx.Value(), "and is handed the backlog")
				},
			},
			{
				Method: "Subscribe",
				Name:   "delivers-to-an-attached-subscriber",
				Claim:  "Subscribe delivers to a subscriber that was already attached",
				Run: func(tb testing.TB, s outbox.Contract, fx outboxtest.ContractFixture) {
					tb.Helper()
					stream, err := s.Subscribe(tb.Context())
					testkit.NoError(tb, err, "a subscriber attaches")

					testkit.NoError(tb, s.Append(tb.Context(), fx.ValueOther()),
						"a record appended after the subscriber attached is accepted")
					testkit.Equal(tb, <-stream, fx.ValueOther(), "and reaches them")
				},
			},
			{
				Method: "Append",
				Name:   "reports-an-unreachable-subscriber",
				Claim:  "Append reports a subscriber it can no longer reach",
				Run: func(tb testing.TB, s outbox.Contract, fx outboxtest.ContractFixture) {
					tb.Helper()
					// A silent drop is the one failure this whole fixture is
					// about, so the drop has to be reported — and a report
					// nothing reaches is a report nothing proves. The reader
					// never reads, which is what puts the subscriber far enough
					// behind.
					_, err := s.Subscribe(tb.Context())
					testkit.NoError(tb, err, "a subscriber attaches")

					for range 32 {
						if err := s.Append(tb.Context(), fx.Value()); err != nil {
							testkit.ErrorIs(tb, err, outboxtest.ErrFull,
								"the append says why it could not be taken")
							return
						}
					}
					tb.Fatalf("a subscriber that never reads was never reported as behind")
				},
			},
			{
				Method: "Subscribe",
				Name:   "refuses-a-backlog-it-cannot-hand-over",
				Claim:  "Subscribe refuses a subscriber it cannot hand the backlog to",
				Run: func(tb testing.TB, s outbox.Contract, fx outboxtest.ContractFixture) {
					tb.Helper()
					// The backlog is loaded at subscribe time, so a log longer
					// than a subscriber can hold is refused rather than
					// truncated — which would lose exactly the records an outbox
					// exists to keep.
					for range 32 {
						if err := s.Append(tb.Context(), fx.Value()); err != nil {
							tb.Fatalf("appending with nobody listening must not fail: %v", err)
						}
					}
					_, err := s.Subscribe(tb.Context())
					testkit.ErrorIs(tb, err, outboxtest.ErrFull,
						"a backlog too long to hand over is refused rather than truncated")
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

	outboxtest.RunContract(t,
		outboxtest.ContractHarness[*outboxtest.InMemory]{Name: "in-memory", New: outboxtest.NewInMemory},
		outboxtest.ContractSuite.Without(outboxtest.ContractSuite.Checks.Append.Smoke()),
	)
}
