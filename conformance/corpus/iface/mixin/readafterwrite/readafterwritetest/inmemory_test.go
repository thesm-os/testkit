// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package readafterwritetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readafterwrite"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readafterwrite/readafterwritetest"
)

// readafterwrite is the model tier's — AUTO-READ-AFTER-WRITE states it — so the
// suite generates the signature family alone, even though the mixin names its
// partner through `write=Write`.
//
// Naming the partner is what makes the law bindable; stating it needs a
// reference to compare against, which is what separates the two tiers.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	fixture := readafterwritetest.DefaultMixedFixture()

	readafterwritetest.AssertMixedContract(t,
		readafterwritetest.MixedSubject("in-memory", func() readafterwrite.Mixed {
			return readafterwritetest.NewInMemory()
		}),
		// The model tier: random sequences against the derived reference,
		// reporting under "model" beside the per-method checks.
		readafterwritetest.MixedModel(),
		readafterwritetest.MixedSeed(func(ctx context.Context, subject readafterwrite.Mixed) error {
			return subject.Write(ctx, fixture.Key, fixture.Value)
		}),
		readafterwritetest.MixedOnRead("returns what was written", func(
			tb testing.TB, subject readafterwrite.Mixed, key string,
		) {
			tb.Helper()
			got, err := subject.Read(tb.Context(), key)
			testkit.NoError(tb, err, "a written key is found")
			testkit.Equal(tb, got, fixture.Value, "and carries what was written")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	readafterwritetest.AssertMixedContract(t,
		readafterwritetest.MixedSubject("in-memory", func() readafterwrite.Mixed {
			return readafterwritetest.NewInMemory()
		}),
		readafterwritetest.MixedWithout("Write/smoke"),
		readafterwritetest.MixedWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestMixedSaturation(t *testing.T) {
	t.Parallel()
	readafterwritetest.MixedModelSaturation(t, func() readafterwrite.Mixed {
		return readafterwritetest.NewInMemory()
	})
}
