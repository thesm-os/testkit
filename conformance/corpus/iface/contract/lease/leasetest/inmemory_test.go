// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package leasetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/lease"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/lease/leasetest"
	"go.thesmos.sh/testkit/engine/model"
)

// lease is the model tier's under ADR-0018: `AUTO-LEASE-DOUBLE-ACQUIRE-BLOCKS`
// and `AUTO-LEASE-RELEASED-ON-CANCEL` state it.
//
// Acquire is classified writer, so the harness seeds every fresh subject
// through it — which means each check meets a subject already holding the
// derived key. That is what the smoke check runs against, and a subject that
// panicked on a second acquire rather than refusing it fails there.
func TestContractContract(t *testing.T) {
	t.Parallel()

	leasetest.AssertContractContract(t,
		leasetest.ContractModel(
			// Freeness probed the only way the interface offers: a key that
			// can be acquired was free, and the probe releases what it took.
			leasetest.ContractModelFree(func(t *model.T, subject lease.Contract, k string) bool {
				if err := subject.Acquire(t.Context(), k); err != nil {
					return false
				}
				_ = subject.Release(t.Context(), k)
				return true
			}),
		),
		leasetest.ContractSubject("in-memory", func() lease.Contract {
			return leasetest.NewInMemory()
		}),
		leasetest.ContractOnAcquire("refuses a key the seed already took", func(
			tb testing.TB, subject lease.Contract, key string,
		) {
			tb.Helper()
			testkit.ErrorIs(tb, subject.Acquire(tb.Context(), key), lease.ErrHeld,
				"a held lease is refused rather than granted twice")
			testkit.NoError(tb, subject.Acquire(tb.Context(), key+"-free"),
				"and a key nobody holds is still available")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	leasetest.AssertContractContract(t,
		leasetest.ContractSubject("in-memory", func() lease.Contract {
			return leasetest.NewInMemory()
		}),
		leasetest.ContractWithout("Acquire/smoke"),
		leasetest.ContractWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// with the same freeness door armed that arms the tier.
func TestContractSaturation(t *testing.T) {
	t.Parallel()
	leasetest.ContractModelSaturation(t, func() lease.Contract {
		return leasetest.NewInMemory()
	}, leasetest.ContractModelFree(func(t *model.T, subject lease.Contract, k string) bool {
		if err := subject.Acquire(t.Context(), k); err != nil {
			return false
		}
		_ = subject.Release(t.Context(), k)
		return true
	}))
}
