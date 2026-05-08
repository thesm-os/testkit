// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package devicetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/gen/suite/testdata/newshapes"
	"go.thesmos.sh/testkit/gen/suite/testdata/newshapes/devicetest"
	"go.thesmos.sh/testkit/suite"
)

func TestDeviceContract(t *testing.T) {
	t.Parallel()
	factory := func() newshapes.Device { return newshapes.NewInMemoryDevice() }

	devicetest.AssertDeviceContract(t, factory,
		// Mutator: Increment — all assertion primitives.
		devicetest.DeviceOnIncrement(
			suite.AssertMutatorSucceeds[newshapes.Device, int64](1),
			suite.AssertMutatorIdempotent[newshapes.Device, int64](1),
		),
		// ReaderWithBool: Load — all assertion primitives.
		devicetest.DeviceOnLoad(
			suite.AssertReaderWithBoolMissing[newshapes.Device, string, int64]("nonexistent"),
			suite.AssertReaderWithBoolConsistent[newshapes.Device, string, int64]("nonexistent", 3),
		),
		// Lookup: Inspect — all assertion primitives.
		devicetest.DeviceOnInspect(
			suite.AssertLookupMissing[newshapes.Device, string, int64, newshapes.Metadata]("nonexistent"),
		),
		// PoisonAccessor: Err — all assertion primitives.
		devicetest.DeviceOnErr(
			suite.AssertPoisonAccessorNilOnFresh[newshapes.Device](),
			suite.AssertPoisonAccessorConsistent[newshapes.Device](),
		),
	)
}

func BenchmarkDeviceContract(b *testing.B) {
	factory := func() newshapes.Device { return newshapes.NewInMemoryDevice() }

	devicetest.BenchmarkDeviceContract(b, factory,
		// Mutator: Increment — hot-path + allocs gate.
		devicetest.DeviceBenchOnIncrement(
			bench.MutatorHotPath[newshapes.Device, int64](1),
			bench.MutatorAllocsWithin[newshapes.Device, int64](1, 0),
		),
		// ReaderWithBool: Load — hot-path + allocs gate.
		devicetest.DeviceBenchOnLoad(
			bench.ReaderWithBoolHotPath[newshapes.Device, string, int64]("nonexistent"),
			bench.ReaderWithBoolAllocsWithin[newshapes.Device, string, int64]("nonexistent", 0),
		),
		// Lookup: Inspect — hot-path + allocs gate.
		devicetest.DeviceBenchOnInspect(
			bench.LookupHotPath[newshapes.Device, string, int64, newshapes.Metadata]("nonexistent"),
			bench.LookupAllocsWithin[newshapes.Device, string, int64, newshapes.Metadata]("nonexistent", 0),
		),
		// PoisonAccessor: Err — allocs gate.
		devicetest.DeviceBenchOnErr(
			bench.PoisonAccessorAllocsWithin[newshapes.Device](0),
		),
	)
}
