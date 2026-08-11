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

	batchwritertest.AssertContractContract(t,
		batchwritertest.ContractSubject("in-memory", func() batchwriter.Contract {
			return batchwritertest.NewInMemory()
		}),
		batchwritertest.ContractOnPut("refuses a value with nothing to file it under", func(
			tb testing.TB, subject batchwriter.Contract, v batchwriter.Value,
		) {
			tb.Helper()
			// The subject's one way to fail, and `mode=atomic` needs one: "an
			// error leaves observable state unchanged" has no case to observe
			// against a write that always succeeds.
			testkit.Error(tb, subject.Put(tb.Context(), batchwriter.Value{Body: v.Body}),
				"an unkeyed value is refused")
			testkit.NoError(tb, subject.Put(tb.Context(), v),
				"and the store still takes a keyed one")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	batchwritertest.AssertContractContract(t,
		batchwritertest.ContractSubject("in-memory", func() batchwriter.Contract {
			return batchwritertest.NewInMemory()
		}),
		batchwritertest.ContractWithout("Put/smoke"),
		batchwritertest.ContractWithoutDouble(),
	)
}
