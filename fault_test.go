// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
)

func TestFaultInjector(t *testing.T) {
	t.Parallel()

	t.Run("fires on every Nth call", func(t *testing.T) {
		t.Parallel()
		fi := testkit.NewFaultInjector(errors.New("boom"), 3)
		// Calls 1,2 → false; 3 → true; 4,5 → false; 6 → true
		results := make([]bool, 6)
		for i := range 6 {
			results[i] = fi.FaultShouldFire()
		}
		testkit.Equal(t, results, []bool{false, false, true, false, false, true},
			"must fire on 3rd and 6th call")
	})

	t.Run("disabled when n is zero", func(t *testing.T) {
		t.Parallel()
		fi := testkit.NewFaultInjector(errors.New("boom"), 0)
		for range 10 {
			testkit.False(t, fi.FaultShouldFire(), "must never fire when disabled")
		}
	})

	t.Run("disabled when n is negative", func(t *testing.T) {
		t.Parallel()
		fi := testkit.NewFaultInjector(errors.New("boom"), -1)
		testkit.False(t, fi.FaultShouldFire(), "must never fire with negative n")
	})

	t.Run("FaultErr is accessible", func(t *testing.T) {
		t.Parallel()
		err := errors.New("injected")
		fi := testkit.NewFaultInjector(err, 1)
		testkit.True(t, errors.Is(fi.FaultErr, err), "FaultErr must be the configured error")
	})

	t.Run("FaultCount tracks calls", func(t *testing.T) {
		t.Parallel()
		fi := testkit.NewFaultInjector(errors.New("boom"), 5)
		testkit.Equal(t, fi.FaultCount(), 0, "initial count must be zero")
		fi.FaultShouldFire()
		fi.FaultShouldFire()
		testkit.Equal(t, fi.FaultCount(), 2, "must count calls")
	})

	t.Run("FaultReset zeroes counter", func(t *testing.T) {
		t.Parallel()
		fi := testkit.NewFaultInjector(errors.New("boom"), 3)
		fi.FaultShouldFire()
		fi.FaultShouldFire()
		fi.FaultReset()
		testkit.Equal(t, fi.FaultCount(), 0, "must be zero after reset")
		// After reset, next fire should be on 3rd call again.
		testkit.False(t, fi.FaultShouldFire(), "call 1 after reset")
		testkit.False(t, fi.FaultShouldFire(), "call 2 after reset")
		testkit.True(t, fi.FaultShouldFire(), "call 3 after reset must fire")
	})
}
