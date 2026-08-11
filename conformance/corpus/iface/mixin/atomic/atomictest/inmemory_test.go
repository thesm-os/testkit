// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package atomictest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/atomic"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/atomic/atomictest"
)

// atomic is the model tier's under ADR-0018 — AUTO-ATOMIC-WRITE states it — so
// the suite generates the signature family and nothing about atomicity.
//
// That is the assignment working rather than a gap: a property about two
// concurrent callers is not something one fixed call can observe.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	fixture := atomictest.DefaultMixedFixture()

	atomictest.AssertMixedContract(t,
		atomictest.MixedSubject("in-memory", func() atomic.Mixed {
			return atomictest.NewInMemory()
		}),
		atomictest.MixedSeed(func(ctx context.Context, subject atomic.Mixed) error {
			return subject.Write(ctx, fixture.Key, fixture.Left, fixture.Right)
		}),
		atomictest.MixedOnRead("returns both halves as they were written", func(
			tb testing.TB, subject atomic.Mixed, key string,
		) {
			tb.Helper()
			left, right, err := subject.Read(tb.Context(), key)
			testkit.NoError(tb, err, "a written key is found")
			testkit.Equal(tb, left, fixture.Left, "the left half is what was written")
			testkit.Equal(tb, right, fixture.Right, "and so is the right")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	atomictest.AssertMixedContract(t,
		atomictest.MixedSubject("in-memory", func() atomic.Mixed {
			return atomictest.NewInMemory()
		}),
		atomictest.MixedWithout("Write/smoke"),
		atomictest.MixedWithoutDouble(),
	)
}
