// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package singleflighttest_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/singleflight"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/singleflight/singleflighttest"
)

// singleflight is the model tier's under ADR-0018:
// `AUTO-SINGLEFLIGHT-COALESCES` states it, and states it with real contention —
// the law launches its own concurrent callers around the compute it supplies.
//
// The check below is the deterministic complement: with a memoizing subject,
// one computation per key is a fact, not a race to observe, so a hand-driven
// pair of callers can assert the exact count the law bounds.
func TestContractContract(t *testing.T) {
	t.Parallel()

	singleflighttest.AssertContractContract(t,
		singleflighttest.ContractModel(),
		singleflighttest.ContractSubject("in-memory", func() singleflight.Contract {
			return singleflighttest.NewInMemory()
		}),
		singleflighttest.ContractOnRun("computes once per key and shares the answer", func(
			tb testing.TB, subject singleflight.Contract, _ string, _ func() string,
		) {
			tb.Helper()
			var calls atomic.Int64
			compute := func() string {
				calls.Add(1)
				return "computed"
			}

			const callers = 4
			var wg sync.WaitGroup
			for range callers {
				wg.Go(func() {
					got, err := subject.Run(tb.Context(), "b6-shared", compute)
					testkit.NoError(tb, err, "a caller is answered")
					testkit.Equal(tb, got, "computed", "with the shared answer")
				})
			}
			wg.Wait()

			testkit.Equal(tb, calls.Load(), int64(1),
				"four callers for one key ran the compute once")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	singleflighttest.AssertContractContract(t,
		singleflighttest.ContractSubject("in-memory", func() singleflight.Contract {
			return singleflighttest.NewInMemory()
		}),
		singleflighttest.ContractWithout("Run/smoke"),
		singleflighttest.ContractWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	singleflighttest.ContractModelSaturation(t, func() singleflight.Contract {
		return singleflighttest.NewInMemory()
	})
}
