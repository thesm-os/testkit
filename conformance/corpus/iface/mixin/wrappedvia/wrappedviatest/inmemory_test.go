// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package wrappedviatest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/wrappedvia"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/wrappedvia/wrappedviatest"
)

// wrappedvia names its partner through `fn=Cause`, and its check is still not
// generated — for a reason the subject makes plain, and which the header now
// records as a refusal.
//
// The claim is that what Open returns unwraps to what Cause reports. Reaching
// it needs an input Open *fails* for, and nothing states which input that is:
// the fixture's alternate is "a value that should not be found", which is a
// different claim from "a value this method refuses".
func TestMixedContract(t *testing.T) {
	t.Parallel()

	wrappedviatest.RunMixed(t,
		wrappedviatest.MixedHarness[*wrappedviatest.InMemory]{Name: "in-memory", New: wrappedviatest.NewInMemory},
		wrappedviatest.MixedChecks{
			{
				Method: "Open",
				Name:   "wraps-the-cause",
				Claim:  "Open wraps the cause it reports",
				Run: func(tb testing.TB, s wrappedvia.Mixed, fx wrappedviatest.MixedFixture) {
					tb.Helper()
					// The failing name is the subject's, not the fixture's:
					// which input a method refuses is exactly what no
					// signature says.
					err := s.Open(tb.Context(), wrappedviatest.FailingName())
					testkit.ErrorIs(tb, err, wrappedviatest.ErrUnderlying,
						"what Open returns unwraps to the cause")
					testkit.ErrorIs(tb, s.Cause(tb.Context()), wrappedviatest.ErrUnderlying,
						"and Cause reports the same one")
				},
			},
		},
	)
}

// Wrapping rather than replacing is the whole point: a subject returning the
// cause bare satisfies errors.Is and loses the context of which call failed.
func TestOpenWrapsRatherThanReplacing(t *testing.T) {
	t.Parallel()

	s := wrappedviatest.NewInMemory()
	err := s.Open(t.Context(), wrappedviatest.FailingName())

	testkit.ErrorIs(t, err, wrappedviatest.ErrUnderlying, "the cause is reachable")
	testkit.False(t, err == wrappedviatest.ErrUnderlying, //nolint:errorlint // identity is the check
		"but the error is not the cause itself, or the call site is lost")
	testkit.Assert(t, err.Error()).Contains(wrappedviatest.FailingName(),
		"and it names what was being opened")
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	wrappedviatest.RunMixed(t,
		wrappedviatest.MixedHarness[*wrappedviatest.InMemory]{Name: "in-memory", New: wrappedviatest.NewInMemory},
		wrappedviatest.MixedSuite.Without(wrappedviatest.MixedSuite.Checks.Open.Smoke()),
	)
}
