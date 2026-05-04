// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package machinetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/model/testdata/newshapes"
	"go.thesmos.sh/testkit/gen/model/testdata/newshapes/machinetest"
)

func TestInMemoryMachineModel(t *testing.T) {
	t.Parallel()

	t.Run("all new shapes detected", func(t *testing.T) {
		t.Parallel()
		// Machine has Mutator (Fold), ReaderWithBool (Lookup),
		// Pure (State), PoisonAccessor (Err).
		// Tier 1: consumer supplies reference (non-CRUD interface).
		machinetest.AssertMachineModel(t,
			func() newshapes.Machine { return newshapes.NewInMemoryMachine() },
			machinetest.MachineModelReference(func() newshapes.Machine {
				return newshapes.NewInMemoryMachine()
			}),
		)
	})
}
