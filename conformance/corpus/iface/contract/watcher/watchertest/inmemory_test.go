// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package watchertest_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/watcher"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/watcher/watchertest"
)

// none is how long a check waits to prove nothing arrives. Short on
// purpose: the claim is absence, and absence does not get truer with time.
const none = 50 * time.Millisecond

// watcher is the model tier's under ADR-0018:
// `AUTO-WATCHER-RETURNS-ON-CHANGE` states it, driving the subscription's
// next= and stop= members the directive names.
//
// Every claim below is statable through the interface, so every one is a row
// rather than a test in this package: a row runs against each subject a
// consumer declares and again through the double, and a package test runs
// against the one implementation this package holds.
func TestContractContract(t *testing.T) {
	t.Parallel()

	watchertest.RunContract(t,
		watchertest.ContractHarness[*watchertest.InMemory]{Name: "in-memory", New: watchertest.NewInMemory},
		// Watch is reader-shaped and Trigger writes, so the rules derive a miss
		// — and a watch has no miss. Attaching to a key nothing has written yet
		// is the ordinary case, not the failing one, so the subject answers a
		// live subscription and the derived claim is wrong for it rather than
		// the other way round.
		watchertest.ContractSuite.Without(watchertest.ContractSuite.Checks.Watch.Miss()),
		watchertest.ContractChecks{
			{
				Method: "Trigger",
				Name:   "wakes-a-watcher-of-the-changed-key",
				Claim:  "Trigger wakes a watcher of the key that changed",
				Run: func(tb testing.TB, s watcher.Contract, fx watchertest.ContractFixture) {
					tb.Helper()
					sub, err := s.Watch(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a watcher attaches")
					defer sub.Stop()

					testkit.NoError(tb, s.Trigger(tb.Context(), fx.Key(), fx.Value()),
						"the change is recorded")
					got, ok := sub.Next(time.Second)
					testkit.True(tb, ok, "and reaches the watcher of that key")
					testkit.Equal(tb, got, fx.Value(), "carrying what was written")
				},
			},
			{
				Method: "Watch",
				Name:   "does-not-wake-a-watcher-of-another-key",
				Claim:  "Watch does not wake a watcher of another key",
				Run: func(tb testing.TB, s watcher.Contract, fx watchertest.ContractFixture) {
					tb.Helper()
					// A subject notifying every watcher on every change
					// satisfies the row above and wakes the whole system on one
					// write.
					sub, err := s.Watch(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a watcher attaches to one key")
					defer sub.Stop()

					testkit.NoError(tb,
						s.Trigger(tb.Context(), fx.KeyOther(), watcher.Value{Key: fx.KeyOther()}),
						"a change to another key is recorded")
					_, ok := sub.Next(none)
					testkit.False(tb, ok, "and does not reach them")
				},
			},
			{
				Method: "Watch",
				Name:   "wakes-every-watcher-of-one-key",
				Claim:  "Watch wakes every watcher of one key",
				Run: func(tb testing.TB, s watcher.Contract, fx watchertest.ContractFixture) {
					tb.Helper()
					// A subject handing each change to whichever watcher it
					// reached first satisfies a one-watcher check and loses half
					// the wake-ups.
					first, err := s.Watch(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "the first watcher attaches")
					defer first.Stop()
					second, err := s.Watch(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "and so does the second")
					defer second.Stop()

					testkit.NoError(tb, s.Trigger(tb.Context(), fx.Key(), fx.Value()),
						"the change is recorded")
					_, ok := first.Next(time.Second)
					testkit.True(tb, ok, "reaching the first")
					_, ok = second.Next(time.Second)
					testkit.True(tb, ok, "and the second")
				},
			},
			{
				Method: "Watch",
				Name:   "delivers-nothing-that-predates-it",
				Claim:  "Watch delivers nothing that predates the watch",
				Run: func(tb testing.TB, s watcher.Contract, fx watchertest.ContractFixture) {
					tb.Helper()
					// A subject keeping a backlog would hand this watcher a
					// change from before it existed — which is `outbox`, and a
					// different contract.
					testkit.NoError(tb, s.Trigger(tb.Context(), fx.Key(), fx.Value()),
						"a change is recorded with nobody watching")

					sub, err := s.Watch(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a watcher attaches afterwards")
					defer sub.Stop()
					_, ok := sub.Next(none)
					testkit.False(tb, ok, "and is handed nothing that predates it")
				},
			},
			{
				Method: "Trigger",
				Name:   "reports-an-unreachable-watcher",
				Claim:  "Trigger reports a watcher it can no longer reach",
				Run: func(tb testing.TB, s watcher.Contract, fx watchertest.ContractFixture) {
					tb.Helper()
					sub, err := s.Watch(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a watcher attaches")
					defer sub.Stop()

					for range 32 {
						if err := s.Trigger(tb.Context(), fx.Key(), fx.Value()); err != nil {
							testkit.ErrorIs(tb, err, watchertest.ErrFull,
								"the trigger says why the change could not be taken")
							return
						}
					}
					tb.Fatalf("a watcher that never reads was never reported as behind")
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

	watchertest.RunContract(t,
		watchertest.ContractHarness[*watchertest.InMemory]{Name: "in-memory", New: watchertest.NewInMemory},
		watchertest.ContractSuite.Without(
			watchertest.ContractSuite.Checks.Watch.Smoke(),
			// The same drop the run above makes, for the same reason.
			watchertest.ContractSuite.Checks.Watch.Miss(),
		),
	)
}
