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
// Every claim below is statable through the interface, so every one is a check
// rather than a test in this package: a check runs against each subject a
// consumer declares and again through the double, and a package test runs
// against the one implementation this package holds.
//
// The fixture is replaced because the derived alternate is a plausible key and
// this subject accepts every plausible key. Watch's "an error carries the zero
// value" check reaches its failure through the alternate, so with a key nothing
// refuses it would fatal against a correct implementation — the empty key is
// one no watcher can serve.
func TestContractContract(t *testing.T) {
	t.Parallel()

	watchertest.AssertContractContract(t,
		watchertest.ContractModel(),
		watchertest.ContractSubject("in-memory", func() watcher.Contract {
			return watchertest.NewInMemory()
		}),
		watchertest.ContractWithFixture(watchertest.ContractFixture{
			Key:      "test-key",
			KeyOther: "",
		}),
		watchertest.ContractOnTrigger("wakes a watcher of the key that changed", func(
			tb testing.TB, subject watcher.Contract, key string, v watcher.Value,
		) {
			tb.Helper()
			sub, err := subject.Watch(tb.Context(), key)
			testkit.NoError(tb, err, "a watcher attaches")
			defer sub.Stop()

			testkit.NoError(tb, subject.Trigger(tb.Context(), key, v), "the change is recorded")
			got, ok := sub.Next(time.Second)
			testkit.True(tb, ok, "and reaches the watcher of that key")
			testkit.Equal(tb, got, v, "carrying what was written")
		}),
		watchertest.ContractOnWatch("does not wake a watcher of another key", func(
			tb testing.TB, subject watcher.Contract, key string,
		) {
			tb.Helper()
			// A subject notifying every watcher on every change satisfies the
			// check above and wakes the whole system on one write.
			sub, err := subject.Watch(tb.Context(), key)
			testkit.NoError(tb, err, "a watcher attaches to one key")
			defer sub.Stop()

			testkit.NoError(tb,
				subject.Trigger(tb.Context(), key+"-other", watcher.Value{Key: key + "-other"}),
				"a change to another key is recorded")
			_, ok := sub.Next(none)
			testkit.False(tb, ok, "and does not reach them")
		}),
		watchertest.ContractOnWatch("wakes every watcher of one key", func(
			tb testing.TB, subject watcher.Contract, key string,
		) {
			tb.Helper()
			// A subject handing each change to whichever watcher it reached
			// first satisfies a one-watcher check and loses half the wake-ups.
			first, err := subject.Watch(tb.Context(), key)
			testkit.NoError(tb, err, "the first watcher attaches")
			defer first.Stop()
			second, err := subject.Watch(tb.Context(), key)
			testkit.NoError(tb, err, "and so does the second")
			defer second.Stop()

			v := watcher.Value{Key: key, Body: "changed"}
			testkit.NoError(tb, subject.Trigger(tb.Context(), key, v), "the change is recorded")
			_, ok := first.Next(time.Second)
			testkit.True(tb, ok, "reaching the first")
			_, ok = second.Next(time.Second)
			testkit.True(tb, ok, "and the second")
		}),
		watchertest.ContractOnWatch("delivers nothing that predates the watch", func(
			tb testing.TB, subject watcher.Contract, key string,
		) {
			tb.Helper()
			// The harness seeds through Trigger, so a subject keeping a backlog
			// would hand this watcher a change from before it existed — which
			// is `outbox`, and a different contract.
			sub, err := subject.Watch(tb.Context(), key)
			testkit.NoError(tb, err, "a watcher attaches after the seed's change")
			defer sub.Stop()
			_, ok := sub.Next(none)
			testkit.False(tb, ok, "and is handed nothing that predates it")
		}),
		watchertest.ContractOnTrigger("reports a watcher it can no longer reach", func(
			tb testing.TB, subject watcher.Contract, key string, v watcher.Value,
		) {
			tb.Helper()
			sub, err := subject.Watch(tb.Context(), key)
			testkit.NoError(tb, err, "a watcher attaches")
			defer sub.Stop()

			for range 32 {
				if err := subject.Trigger(tb.Context(), key, v); err != nil {
					testkit.ErrorIs(tb, err, watchertest.ErrFull,
						"the trigger says why the change could not be taken")
					return
				}
			}
			tb.Fatalf("a watcher that never reads was never reported as behind")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	watchertest.AssertContractContract(t,
		watchertest.ContractSubject("in-memory", func() watcher.Contract {
			return watchertest.NewInMemory()
		}),
		// The override travels with the subject rather than with the run above:
		// Watch's miss check needs a key this implementation refuses, and every
		// run of it needs one. Leaving it off here fails with the message that
		// names the fix, which is how the omission was found.
		watchertest.ContractWithFixture(watchertest.ContractFixture{
			Key:      "test-key",
			KeyOther: "",
		}),
		watchertest.ContractWithout("Watch/smoke"),
		watchertest.ContractWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	watchertest.ContractModelSaturation(t, func() watcher.Contract {
		return watchertest.NewInMemory()
	})
}
