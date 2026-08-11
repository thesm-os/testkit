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

// The write is visible to the very next read, with nothing in between. An
// implementation buffering writes satisfies every generated check and fails
// this one.
func TestWriteIsVisibleImmediately(t *testing.T) {
	t.Parallel()

	s := readafterwritetest.NewInMemory()
	testkit.NoError(t, s.Write(t.Context(), "k", "v"), "the write succeeds")

	got, err := s.Read(t.Context(), "k")
	testkit.NoError(t, err, "and the next read finds it")
	testkit.Equal(t, got, "v", "carrying what was written")
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
