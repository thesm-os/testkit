// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import (
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/suite"
)

// LegOf answers which leg carries one law's evidence: the class it
// reports under, and whether it rides its own leg. The clocked family
// derives from the bindings' own Timeaware fact; the remaining
// own-leg laws are catalogue rows — legs needing what the bundled
// observational run cannot provide: a lifecycle probe, a poisoned
// subject, writes of their own. Everything else bundles under the
// shared sequences, which makes the answer total over any registry:
// an unlisted law is an observational law by default, and a law that
// needed its own leg shows up as a bundle row the corpus locks
// contradict — the parity gate's to catch, loudly.
//
// In this package rather than the suite generator because leg
// assignment is a tier fact with the same three readers the rest of
// the catalogue has: the suite tier plans the rows, the model tier
// will emit the legs, and the conformance gate holds the table to the
// law registry.
func LegOf(law string) (suite.Class, bool) {
	if b, ok := BindingFor(law); ok && b.Timeaware {
		return suite.ClassClocked, true
	}
	class, own := ownLegs()[law]
	return class, own
}

// ownLegs is the non-clocked own-leg table. A row here is a design
// event; the census holds every key to the live law registry.
func ownLegs() map[string]suite.Class {
	return map[string]suite.Class{
		lawid.LifecycleAfterClose:      suite.ClassLifecycle,
		lawid.PoisonConsistent:         suite.ClassPoison,
		lawid.AppenderMonotonicOffsets: suite.ClassAppender,
		lawid.CursorCloseIdempotent:    suite.ClassLifecycle,
		lawid.CursorNextAfterClose:     suite.ClassLifecycle,
		lawid.LeaseDoubleAcquireBlocks: suite.ClassLaws,
		lawid.LeaseReleasedOnCancel:    suite.ClassConcurrent,
	}
}
