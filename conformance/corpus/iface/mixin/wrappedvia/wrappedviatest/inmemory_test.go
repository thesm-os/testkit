// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package wrappedviatest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/wrappedvia"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/wrappedvia/wrappedviatest"
)

// wrappedvia is the suite tier's and names its partner through `fn=Cause`, and
// its check is still not generated — for a reason the subject makes plain.
//
// The claim is that what Open returns unwraps to what Cause reports. Reaching
// it needs an input Open *fails* for, and nothing states which input that is:
// the fixture's alternate is "a value that should not be found", which is a
// different claim from "a value this method refuses". That is the same gap
// `errors` has, and the same one `partition` needed an axis to close.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	wrappedviatest.AssertMixedContract(t,
		wrappedviatest.MixedModel(),
		wrappedviatest.MixedSubject("in-memory", func() wrappedvia.Mixed {
			return wrappedviatest.NewInMemory()
		}),
		wrappedviatest.MixedOnOpen("wraps the cause it reports", func(
			tb testing.TB, subject wrappedvia.Mixed, name string,
		) {
			tb.Helper()
			err := subject.Open(tb.Context(), wrappedviatest.FailingName())
			testkit.ErrorIs(tb, err, wrappedviatest.ErrUnderlying,
				"what Open returns unwraps to the cause")
			testkit.ErrorIs(tb, subject.Cause(tb.Context()), wrappedviatest.ErrUnderlying,
				"and Cause reports the same one")
		}),
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

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	wrappedviatest.AssertMixedContract(t,
		wrappedviatest.MixedSubject("in-memory", func() wrappedvia.Mixed {
			return wrappedviatest.NewInMemory()
		}),
		wrappedviatest.MixedWithout("Open/smoke"),
		wrappedviatest.MixedWithoutDouble(),
	)
}
