// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"
	"testing"
)

// Bundle accumulates what a run's options build — subjects, extra checks,
// drops — and every wiring error met on the way. A generated run surface
// embeds one per entry point, so the collect-then-report shape has one
// home: every mistake is reported in one pass rather than one per rerun,
// and nothing executes over a partially wired run.
type Bundle[S any] struct {
	// Subjects are the lowered harnesses the run will drive.
	Subjects []Subject[S]
	// Extra holds the consumer's own bound rows, run beside the
	// generated checks.
	Extra []Check[S]
	// Drops collects the IDs the run was told to skip; validated
	// against the check set before anything runs.
	Drops []ID

	errs   []error
	cfgSet bool
}

// ConfigOnce reports whether this is the run's first config, and
// records the duplicate as a wiring error otherwise: every other
// ambiguity in the option surface fails loudly, and a silently
// discarded first config is a run measuring pools nobody chose.
func (b *Bundle[S]) ConfigOnce(name string) bool {
	if b.cfgSet {
		b.errs = append(b.errs, fmt.Errorf(
			"two %ss passed; the second would silently win — pass at most one", name))
		return false
	}
	b.cfgSet = true
	return true
}

// AddSubject records a lowered subject, or the harness misconfiguration
// that prevented the lowering.
func (b *Bundle[S]) AddSubject(sub Subject[S], err error) {
	if err != nil {
		// The ordinal identifies a harness whose misconfiguration is
		// exactly that it carries no name yet: "subject 3" is findable
		// in an options list; "a harness" is not.
		b.errs = append(b.errs, fmt.Errorf("subject %d: %w", len(b.Subjects)+len(b.errs)+1, err))
		return
	}
	b.Subjects = append(b.Subjects, sub)
}

// AddCheck records a bound row, or the row mistake that prevented the
// binding.
func (b *Bundle[S]) AddCheck(c Check[S], err error) {
	if err != nil {
		b.errs = append(b.errs, err)
		return
	}
	b.Extra = append(b.Extra, c)
}

// AddDrops records IDs the run was told to skip.
func (b *Bundle[S]) AddDrops(ids ...ID) { b.Drops = append(b.Drops, ids...) }

// AddErr records a wiring error with no value attached — a refused
// config, say.
func (b *Bundle[S]) AddErr(err error) { b.errs = append(b.errs, err) }

// Fail reports every collected error under the entry point's name and
// stops the test when there were any. Called between option collection
// and execution: a misused run fails with the full list of fixes and
// runs nothing.
func (b *Bundle[S]) Fail(tb testing.TB, entry string) {
	tb.Helper()
	for _, err := range b.errs {
		tb.Errorf("%s: %v", entry, err)
	}
	if len(b.errs) > 0 {
		tb.FailNow()
	}
}
