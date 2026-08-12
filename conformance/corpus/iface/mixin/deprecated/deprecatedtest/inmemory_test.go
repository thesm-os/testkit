// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package deprecatedtest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deprecated"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deprecated/deprecatedtest"
)

// deprecated generates no check of its own, and that is the decision rather
// than an omission.
//
// A deprecated method keeps every obligation it had until it is deleted, so a
// check that skipped it would stop testing a method still in use, and one that
// merely announced the deprecation would assert nothing. Both spellings are
// held to the full signature family instead.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	fixture := deprecatedtest.DefaultMixedFixture()

	deprecatedtest.AssertMixedContract(t,
		deprecatedtest.MixedModel(),
		deprecatedtest.MixedSubject("in-memory", func() deprecated.Mixed {
			return deprecatedtest.NewInMemory()
		}),
		deprecatedtest.MixedSeed(func(_ context.Context, subject deprecated.Mixed) error {
			subject.(*deprecatedtest.InMemory).Put(fixture.Key, "stored")
			return nil
		}),
		deprecatedtest.MixedOnOld("answers as the replacement does", func(
			tb testing.TB, subject deprecated.Mixed, key string,
		) {
			tb.Helper()
			// The only claim worth making about a deprecated method: that it
			// has not quietly diverged from what replaced it.
			old, oldErr := subject.Old(tb.Context(), key)
			replacement, newErr := subject.New(tb.Context(), key)
			testkit.NoError(tb, oldErr, "the deprecated spelling still works")
			testkit.NoError(tb, newErr, "and so does the replacement")
			testkit.Equal(tb, old, replacement, "and the two agree")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	deprecatedtest.AssertMixedContract(t,
		deprecatedtest.MixedSubject("in-memory", func() deprecated.Mixed {
			return deprecatedtest.NewInMemory()
		}),
		deprecatedtest.MixedWithout("Old/smoke"),
		deprecatedtest.MixedWithoutDouble(),
	)
}
