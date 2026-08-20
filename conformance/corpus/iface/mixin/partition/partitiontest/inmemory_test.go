// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package partitiontest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/partition"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/partition/partitiontest"
)

// `//testkit:mixin partition read=Read axis=partition` is the wiring: read
// names what observes the isolation, axis names the parameter to vary.
//
// The isolation itself is the model tier's. What the row states is the miss
// beside it: two writes that never collide would pass an isolation check
// against a store that answers for every key, so somebody has to ask what
// happens to a key nobody wrote.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	partitiontest.RunMixed(t,
		partitiontest.MixedHarness[*partitiontest.InMemory]{Name: "in-memory", New: partitiontest.NewInMemory},
		partitiontest.MixedChecks{
			{
				Method: "Read",
				Name:   "misses-a-key-its-partition-lacks",
				Claim:  "Read reports a key its partition does not hold",
				Run: func(tb testing.TB, s partition.Mixed, fx partitiontest.MixedFixture) {
					tb.Helper()
					// The partition is written first, so the miss below is
					// about the key rather than about an empty store.
					testkit.NoError(tb, s.Put(tb.Context(), fx.Partition(), fx.Key(), fx.Value()),
						"the key is written into its partition")

					got, err := s.Read(tb.Context(), fx.Partition(), fx.Key())
					testkit.NoError(tb, err, "and reads back from it")
					testkit.Equal(tb, got, fx.Value(), "carrying what was written")

					// A store answering for a key nobody wrote is
					// indistinguishable from one that found it.
					_, err = s.Read(tb.Context(), fx.Partition(), fx.KeyOther())
					testkit.Error(tb, err, "an unwritten key is a miss")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	partitiontest.RunMixed(t,
		partitiontest.MixedHarness[*partitiontest.InMemory]{Name: "in-memory", New: partitiontest.NewInMemory},
		partitiontest.MixedSuite.Without(partitiontest.MixedSuite.Checks.Put.Smoke()),
	)
}
