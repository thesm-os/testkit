// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package idempotenttest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotent"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotent/idempotenttest"
)

// idempotent is the model tier's — AUTO-IDEMPOTENT-WRITE states it — so the
// suite generates the signature family alone.
//
// That is the assignment working rather than a gap: the repeat write and the
// single write return the same thing, so no one call can tell them apart.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	fixture := idempotenttest.DefaultMixedFixture()

	idempotenttest.AssertMixedContract(t,
		idempotenttest.MixedSubject("in-memory", func() idempotent.Mixed {
			return idempotenttest.NewInMemory()
		}),
		idempotenttest.MixedSeed(func(ctx context.Context, subject idempotent.Mixed) error {
			return subject.Put(ctx, fixture.Key, fixture.Value)
		}),
		idempotenttest.MixedOnRead("returns what was written", func(
			tb testing.TB, subject idempotent.Mixed, key string,
		) {
			tb.Helper()
			got, err := subject.Read(tb.Context(), key)
			testkit.NoError(tb, err, "a written key is found")
			testkit.Equal(tb, got, fixture.Value, "and carries what was written")
		}),
	)
}

// Repeating a write leaves the store as one write did, which the interface
// cannot show: Put reports only whether it failed and Read answers alike either
// way. Only the write count separates them.
func TestPutIsIdempotent(t *testing.T) {
	t.Parallel()

	s := idempotenttest.NewInMemory()
	testkit.NoError(t, s.Put(t.Context(), "k", "v"), "the first write succeeds")
	testkit.NoError(t, s.Put(t.Context(), "k", "v"), "and so does the repeat")
	testkit.Equal(t, s.Writes(), 1, "but only one write landed")
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	idempotenttest.AssertMixedContract(t,
		idempotenttest.MixedSubject("in-memory", func() idempotent.Mixed {
			return idempotenttest.NewInMemory()
		}),
		idempotenttest.MixedWithout("Put/smoke"),
		idempotenttest.MixedWithoutDouble(),
	)
}
