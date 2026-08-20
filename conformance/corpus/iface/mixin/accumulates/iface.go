// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package accumulates is the mixin-axis fixture for the accumulates mixin,
// which declares that repeating the call compounds rather than coalescing.
//
// The second position on the effect axis, beside [idempotent] — not its
// negation. A mixin appears only because somebody wrote the directive, so
// absence already means unclaimed and there is nothing to switch off
// (docs/adr/0016). What the pair distinguishes is a state absence cannot:
// "nobody considered whether repeating this is safe" against "it was
// considered, and it compounds".
//
// The interface carries a reader for the same reason idempotent's does: a
// claim about what two calls leave behind needs something to observe it
// with, and a single method would have nothing to compare.
//
// [idempotent]: go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotent
package accumulates

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out accumulatestest/ pkg=accumulatestest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Add compounds. Two calls with one argument leave twice what one call
	// left, which is the whole claim and the opposite of what idempotent's
	// Put asserts about the same shape.
	//testkit:mixin accumulates
	Add(ctx context.Context, key string, amount int) error

	// Total observes what the additions left. Any N ≥ 2 proves the claim,
	// so the mixin takes no count — the number is the check's choice, not
	// the author's assertion.
	Total(ctx context.Context, key string) (int, error)
}
