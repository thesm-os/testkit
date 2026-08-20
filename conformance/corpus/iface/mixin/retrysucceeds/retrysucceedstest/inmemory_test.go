// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package retrysucceedstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/retrysucceeds"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/retrysucceeds/retrysucceedstest"
)

// retrysucceeds is the suite tier's under ADR-0018 and generates no check —
// the header records the gap — because the mixin names no attempt count.
//
// "Succeeds within the declared attempts" is not a statement until a number is
// declared, the same reason `timeout` is gated on its `duration` rather than on
// the mixin. Without one, the only generatable check is "call it and expect
// success", which is the smoke check under another name and would fail this
// subject, whose first attempts are meant to fail.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	retrysucceedstest.RunMixed(
		t,
		retrysucceedstest.MixedHarness[*retrysucceedstest.InMemory]{
			Name: "in-memory",
			New:  retrysucceedstest.NewInMemory,
		},
		// A first call is supposed to fail, so the checks demanding success
		// from one are the ones this subject legitimately violates.
		retrysucceedstest.MixedSuite.Without(retrysucceedstest.MixedSuite.Checks.Call.Smoke()),
		retrysucceedstest.MixedChecks{
			{
				Method: "Call",
				Name:   "succeeds-within-the-retries",
				Claim:  "Call succeeds once a caller retries it",
				Run: func(tb testing.TB, s retrysucceeds.Mixed, fx retrysucceedstest.MixedFixture) {
					tb.Helper()
					// What a caller of this interface writes, and what the
					// mixin promises will terminate.
					testkit.NoError(tb, retryUntilSuccess(tb.Context(), s, fx.Key()),
						"a bounded retry loop gets an answer")

					n, err := s.Attempts(tb.Context())
					testkit.NoError(tb, err, "the attempt count is readable")
					testkit.True(tb, n > 1, "and more than one attempt was needed")
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

	retrysucceedstest.RunMixed(
		t,
		retrysucceedstest.MixedHarness[*retrysucceedstest.InMemory]{
			Name: "in-memory",
			New:  retrysucceedstest.NewInMemory,
		},
		retrysucceedstest.MixedSuite.Without(retrysucceedstest.MixedSuite.Checks.Call.Smoke()),
	)
}

// retryUntilSuccess is what a caller of this interface writes, and what the
// mixin promises will terminate.
//
// Bounded rather than unbounded: a subject that never succeeds is one this loop
// must report rather than hang on, and the bound is the failure this row exists
// to surface.
func retryUntilSuccess(ctx context.Context, subject retrysucceeds.Mixed, key string) error {
	var err error
	for range maxAttempts {
		if err = subject.Call(ctx, key); err == nil {
			return nil
		}
	}
	return err
}

// maxAttempts bounds the retry loop.
const maxAttempts = 8
