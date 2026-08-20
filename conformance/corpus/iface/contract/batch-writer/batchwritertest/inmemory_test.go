// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package batchwritertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	batchwriter "go.thesmos.sh/testkit/conformance/corpus/iface/contract/batch-writer"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/batch-writer/batchwritertest"
)

// batch-writer is the model tier's under ADR-0018, and RFC-0002's table said
// suite. The rule settles it rather than an opinion: `mode=atomic` is the claim
// that an error leaves observable state unchanged, and `AUTO-ATOMIC-WRITE`
// already implements exactly that — snapshot, write, and on failure compare the
// snapshot back.
//
// Discharging it needs two things a fixed sequence against one subject cannot
// produce: an observation of the state, which this contract declares no reader
// role for, and a write that fails on demand. A suite check written without
// them would assert that a successful write succeeded.
func TestContractContract(t *testing.T) {
	t.Parallel()

	batchwritertest.RunContract(t,
		batchwritertest.ContractHarness[*batchwritertest.InMemory]{Name: "in-memory", New: batchwritertest.NewInMemory},
		batchwritertest.ContractChecks{
			{
				Method: "Put",
				Name:   "refuses-unkeyed",
				Claim:  "Put refuses a value with nothing to file it under",
				Run: func(tb testing.TB, s batchwriter.Contract, fx batchwritertest.ContractFixture) {
					tb.Helper()
					// The subject's one way to fail, and `mode=atomic` needs
					// one: "an error leaves observable state unchanged" has no
					// case to observe against a write that always succeeds.
					testkit.Error(tb, s.Put(tb.Context(), batchwriter.Value{Body: fx.Value().Body}),
						"an unkeyed value is refused")
					testkit.NoError(tb, s.Put(tb.Context(), fx.Value()),
						"and the store still takes a keyed one")
				},
			},
			{
				Method: "Put",
				Name:   "a-refused-write-lands-nowhere",
				Claim:  "Put leaves the reader answering as it did when it refuses",
				Run: func(tb testing.TB, s batchwriter.Contract, fx batchwritertest.ContractFixture) {
					tb.Helper()
					// `mode=atomic` read through the role that now exists to
					// read it. Before the reader was declared the only statable
					// claim was that a good write succeeds, which holds for a
					// store that also keeps half a refused one.
					held := fx.Value()
					testkit.NoError(tb, s.Put(tb.Context(), held), "a keyed value lands")

					before, err := s.Get(tb.Context(), held.Key)
					testkit.NoError(tb, err, "and reads back")

					testkit.Error(tb, s.Put(tb.Context(), batchwriter.Value{Body: "unkeyed"}),
						"the unkeyed write is refused")

					after, err := s.Get(tb.Context(), held.Key)
					testkit.NoError(tb, err, "the earlier value is still there")
					testkit.Equal(tb, after, before, "unchanged by the write that failed")
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

	batchwritertest.RunContract(t,
		batchwritertest.ContractHarness[*batchwritertest.InMemory]{Name: "in-memory", New: batchwritertest.NewInMemory},
		batchwritertest.ContractSuite.Without(batchwritertest.ContractSuite.Checks.Put.Smoke()),
	)
}
