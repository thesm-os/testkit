// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package retrysucceedstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/retrysucceeds"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/retrysucceeds/retrysucceedstest"
)

// retrysucceeds is the suite tier's under ADR-0018 and generates no check,
// because the mixin names no attempt count.
//
// "Succeeds within the declared attempts" is not a statement until a number is
// declared — the same reason `timeout` is gated on its `duration` rather than
// on the mixin. Without one, the only generatable check is "call it and expect
// success", which is the smoke check under another name and would fail this
// subject, whose first attempts are meant to fail.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	retrysucceedstest.AssertMixedContract(t,
		retrysucceedstest.MixedModel(),
		retrysucceedstest.MixedSubject("in-memory", func() retrysucceeds.Mixed {
			return retrysucceedstest.NewInMemory()
		}),
		// The derived seed calls Call once, which is supposed to fail here —
		// so the seed retries, which is the mixin restated. Without this every
		// check runs against a subject the harness could not populate.
		retrysucceedstest.MixedSeed(func(ctx context.Context, subject retrysucceeds.Mixed) error {
			return retryUntilSuccess(ctx, subject, "seed")
		}),
		// A first call is supposed to fail, so the checks demanding success
		// from one are the ones this subject legitimately violates.
		retrysucceedstest.MixedWithout(
			"Call/smoke",
			"Call/reports a cancelled context",
			"Call/reports an expired deadline",
		),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	retrysucceedstest.AssertMixedContract(t,
		retrysucceedstest.MixedSubject("in-memory", func() retrysucceeds.Mixed {
			return retrysucceedstest.NewInMemory()
		}),
		retrysucceedstest.MixedSeed(func(ctx context.Context, subject retrysucceeds.Mixed) error {
			return retryUntilSuccess(ctx, subject, "seed")
		}),
		retrysucceedstest.MixedWithout(
			"Call/smoke",
			"Call/reports a cancelled context",
			"Call/reports an expired deadline",
		),
		retrysucceedstest.MixedWithoutDouble(),
	)
}

// retryUntilSuccess is what a caller of this interface writes, and what the
// mixin promises will terminate.
//
// Bounded rather than unbounded: a subject that never succeeds is one this loop
// must report rather than hang on, and the bound is the failure the seed exists
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

// maxAttempts bounds the seed'"'"'s retry loop.
const maxAttempts = 8

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestMixedSaturation(t *testing.T) {
	t.Parallel()
	retrysucceedstest.MixedModelSaturation(t, func() retrysucceeds.Mixed {
		return retrysucceedstest.NewInMemory()
	})
}
