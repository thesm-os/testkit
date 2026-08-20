// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import "testing"

// skipUntilModelRelinked defers a gate whose subject is the model tier.
//
// The tier is unregistered while the suite generator is rewritten — it
// reads the suite's emit node in forty-one places and that node is being
// reshaped — so every gate that measures what the model tier BOUND now
// measures an empty run. Each of these is a correct gate pointed at
// nothing.
//
// A skip rather than a relaxed bound, for the reason the evidence census
// gives one file over: a threshold lowered to what a dormant tier
// produces is a threshold the tree still clears on the morning the tier
// comes back producing nothing. The skip is deferred work with an owner;
// the relaxation is the work forgotten.
//
// The expiry hands the deferral to the skip-expiry gate rather than to
// anyone's memory. Past that date the build reddens here until the tier
// is relinked in [generator.Generators] or the date is argued forward.
func skipUntilModelRelinked(tb testing.TB) {
	tb.Helper()
	tb.Skip("the model tier is unregistered while the suite generator is rewritten, " +
		"so this gate measures an empty run; expires 2026-09-18")
}
