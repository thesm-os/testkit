// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package machinetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/model/testdata/thesmos"
	"go.thesmos.sh/testkit/gen/model/testdata/thesmos/machinetest"
)

func TestMachineModel(t *testing.T) {
	t.Parallel()

	t.Run("counter SUT vs slice ref", func(t *testing.T) {
		t.Parallel()
		// Different impls: InMemoryMachine (counter-based) as SUT,
		// RefMachine (slice-based) as reference. Both should produce
		// identical State/ExpectedSeq/Err after the same Fold sequence.
		machinetest.AssertMachineModel(t,
			func() thesmos.Machine { return thesmos.NewInMemoryMachine() },
			machinetest.MachineModelReference(func() thesmos.Machine {
				return thesmos.NewRefMachine()
			}),
		)
	})

	t.Run("catches broken Fold", func(t *testing.T) {
		t.Parallel()
		// Negative: BrokenFoldMachine increments PatchCount but not Seq.
		// The Pure action on State() compares State.Seq between SUT and
		// ref — the divergence must be caught.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			machinetest.AssertMachineModel(ft,
				func() thesmos.Machine { return thesmos.NewBrokenFoldMachine() },
				machinetest.MachineModelReference(func() thesmos.Machine {
					return thesmos.NewRefMachine()
				}),
			)
		}()
		<-done
		if !ft.Failed() {
			t.Fatal("framework should have caught broken Fold (Seq not incremented)")
		}
	})
}
