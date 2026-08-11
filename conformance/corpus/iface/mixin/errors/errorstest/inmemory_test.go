// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package errorstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/errors"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/errors/errorstest"
)

// The errors mixin is the suite tier's and its check is not generated, because
// the sentinels come from a different directive than the mixin.
//
// `//testkit:fault ErrNotFound ErrGone` names them, and that is the fault
// generator's key — it decides what the double can be told to return. Which
// sentinel a real subject owes for which input is not stated anywhere, so a
// generated check would have to guess, and the guess would be that the first
// declared sentinel is the miss.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	fixture := errorstest.DefaultMixedFixture()

	errorstest.AssertMixedContract(t,
		errorstest.MixedSubject("in-memory", func() errors.Mixed {
			return errorstest.NewInMemory()
		}),
		errorstest.MixedSeed(func(_ context.Context, subject errors.Mixed) error {
			subject.(*errorstest.InMemory).Put(fixture.Key, "stored")
			return nil
		}),
		errorstest.MixedOnGet("reports each declared sentinel for its own case", func(
			tb testing.TB, subject errors.Mixed, key string,
		) {
			tb.Helper()
			_, err := subject.Get(tb.Context(), fixture.KeyOther)
			testkit.ErrorIs(tb, err, errors.ErrNotFound,
				"an absent key is a miss rather than an unlabelled failure")

			_, err = subject.Get(tb.Context(), errorstest.GoneKey())
			testkit.ErrorIs(tb, err, errors.ErrGone,
				"and a removed one is distinguishable from a missing one")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	errorstest.AssertMixedContract(t,
		errorstest.MixedSubject("in-memory", func() errors.Mixed {
			return errorstest.NewInMemory()
		}),
		errorstest.MixedWithout("Get/smoke"),
		errorstest.MixedWithoutDouble(),
	)
}
