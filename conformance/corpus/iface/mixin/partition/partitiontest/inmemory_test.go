// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package partitiontest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/partition"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/partition/partitiontest"
)

// The first generated check that writes twice, and the one that took two
// near-misses to get right.
//
// `//testkit:mixin partition read=Read axis=partition` is the wiring: read names
// what observes the isolation, axis names the parameter to vary. Both are
// necessary. Varying every parameter gives two writes that never collide on a
// key, and holding the payload gives two writes an implementation can clobber
// with an identical value — each version passes against a subject that ignores
// partitions entirely.
//
// What the generated check does instead: hold every identifier the reader
// shares, vary the axis and the payload, read the first partition back.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	partitiontest.AssertMixedContract(t,
		partitiontest.MixedSubject("in-memory", func() partition.Mixed {
			return partitiontest.NewInMemory()
		}),
		// Read is a miss for an unwritten partition, which the derived
		// alternate is — but the zero-value check calls it on a subject the
		// seed never touched, where every partition is unwritten and the miss
		// is the only outcome. That is the check working; it is dropped only
		// because Read has no partition to hit until this suite writes one.
		partitiontest.MixedWithout("Read/an error carries the zero value"),
	)
}

// Isolation is per partition, not per key: a store hashing the two together
// would satisfy the generated check and lose the ability to drop a partition.
func TestPartitionsAreSeparateNamespaces(t *testing.T) {
	t.Parallel()

	s := partitiontest.NewInMemory()
	testkit.NoError(t, s.Put(t.Context(), "a", "k", "one"), "writing to a succeeds")
	testkit.NoError(t, s.Put(t.Context(), "b", "k", "two"), "writing to b succeeds")

	got, err := s.Read(t.Context(), "a", "k")
	testkit.NoError(t, err, "reading a succeeds")
	testkit.Equal(t, got, "one", "and returns a's value")

	_, err = s.Read(t.Context(), "c", "k")
	testkit.ErrorIs(t, err, partitiontest.ErrNotFound,
		"a partition nothing was written to holds nothing")
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	partitiontest.AssertMixedContract(t,
		partitiontest.MixedSubject("in-memory", func() partition.Mixed {
			return partitiontest.NewInMemory()
		}),
		partitiontest.MixedWithout(
			"Put/smoke",
			"Read/an error carries the zero value",
		),
		partitiontest.MixedWithoutDouble(),
	)
}
