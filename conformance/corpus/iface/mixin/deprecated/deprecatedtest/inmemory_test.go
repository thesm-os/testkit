// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package deprecatedtest_test

import (
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

	fx := deprecatedtest.DefaultMixedFixture()

	deprecatedtest.RunMixed(t,
		deprecatedtest.MixedHarness[*deprecatedtest.InMemory]{
			Name: "in-memory",
			// Nothing on this interface writes, so the seed is the
			// constructor's: both spellings read, neither stores.
			New: func() *deprecatedtest.InMemory {
				s := deprecatedtest.NewInMemory()
				s.Put(fx.Key(), "stored")
				return s
			},
		},
		deprecatedtest.MixedChecks{
			{
				Method: "Old",
				Name:   "agrees-with-the-replacement",
				Claim:  "Old answers as the replacement does",
				Run: func(tb testing.TB, s deprecated.Mixed, fx deprecatedtest.MixedFixture) {
					tb.Helper()
					// The only claim worth making about a deprecated method:
					// that it has not quietly diverged from what replaced it.
					old, oldErr := s.Old(tb.Context(), fx.Key())
					replacement, newErr := s.New(tb.Context(), fx.Key())
					testkit.NoError(tb, oldErr, "the deprecated spelling still works")
					testkit.NoError(tb, newErr, "and so does the replacement")
					testkit.Equal(tb, old, replacement, "and the two agree")
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

	deprecatedtest.RunMixed(t,
		deprecatedtest.MixedHarness[*deprecatedtest.InMemory]{Name: "in-memory", New: deprecatedtest.NewInMemory},
		deprecatedtest.MixedSuite.Without(deprecatedtest.MixedSuite.Checks.Old.Smoke()),
	)
}
