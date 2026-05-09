// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"errors"
	"fmt"
	"testing"

	"go.thesmos.sh/testkit"
)

// assertSentinelMatch fails the test when err doesn't match any of
// the declared sentinels via [errors.Is]. The sentinel-bearing
// runtime helpers (AssertWriteRejectInvalid, AssertReturnsSentinel,
// etc.) take their sentinels variadically so a method that declares
// multiple //testkit:errors entries can assert "one of these
// sentinels was returned" — under a single zero-valued invalid
// input, only one sentinel will fire on a given impl, but the
// directive's semantics permit any of the declared set.
//
// The single-sentinel case routes through [testkit.ErrorIs] so the
// produced failure message is the same as a hand-written ErrorIs
// call. The multi-sentinel case fails with a list of the declared
// sentinels and the actual error.
func assertSentinelMatch(tb testing.TB, err error, msg string, sentinels ...error) {
	tb.Helper()
	switch len(sentinels) {
	case 0:
		testkit.True(tb, false, msg+" (no sentinels declared)")
	case 1:
		testkit.ErrorIs(tb, err, sentinels[0], msg)
	default:
		for _, s := range sentinels {
			if errors.Is(err, s) {
				return
			}
		}
		testkit.True(tb, false, fmt.Sprintf("%s: got %v, want one of %v", msg, err, sentinels))
	}
}
