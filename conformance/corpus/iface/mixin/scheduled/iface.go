// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package scheduled is the mixin-axis fixture for the scheduled mixin, which
// declares that a task registered for a future instant has run once the clock
// passes it.
//
// The directive names both halves, and the fired-count accessor is the
// load-bearing one: a run that cannot count firings reports every scheduler as
// correct, including one that fires nothing at all.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package scheduled

import (
	"context"
	"time"
)

// Mixed is the fixture interface.
//
// The model tier runs clocked: both methods are the scheduled mixin's own
// partners, and the consumer supplies MixedModelClocked so the property owns
// a TestClock the scheduled-fires law can advance. Unclocked, no sequence
// would ever move time and every firing claim would hold vacuously.
//
//testkit:out scheduledtest/ pkg=scheduledtest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// At registers a task for the given offset from now.
	//testkit:mixin scheduled schedule=At fired=Fired
	At(ctx context.Context, after time.Duration) error

	// Fired counts what has run, which is what makes the claim checkable.
	Fired(ctx context.Context) (int, error)
}
