// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"
)

// Coverage is the difference between what eidos can classify and what the
// corpus actually exercises.
//
// Stamped holds what the annotator produced when run over the corpus; Missing
// holds registered classifications that nothing stamped. A classification is
// covered when the annotator stamped it — not when a directory is named after
// it, because a fixture whose directive reads `idempotant` sits in a correctly
// named directory and stamps nothing.
type Coverage struct {
	// Stamped is what the annotator produced, per axis.
	Stamped map[string][]string

	// Missing is registered-but-unstamped, per axis. Empty means complete.
	Missing map[string][]string
}

// Axis names, used as keys in both maps. They match eidos's package names so a
// reader chasing a gap knows which registry to look in.
const (
	AxisDetector = "detector"
	AxisContract = "contract"
	AxisMixin    = "mixin"
)

// Registered returns every classification eidos ships, per axis.
//
// The lists come from the registries rather than from a checked-in table, so a
// classification added upstream appears here on the next build and starts
// failing the gate until the corpus catches up. That failure is the intended
// signal: it is how the corpus learns the vocabulary grew.
func Registered() map[string][]string {
	out := map[string][]string{
		AxisDetector: make([]string, 0, len(detectors.All())),
		AxisContract: make([]string, 0, len(contracts.All())),
		AxisMixin:    make([]string, 0, len(mixins.All())),
	}
	for _, d := range detectors.All() {
		out[AxisDetector] = append(out[AxisDetector], d.Name)
	}
	for _, c := range contracts.All() {
		out[AxisContract] = append(out[AxisContract], c.Name)
	}
	for _, m := range mixins.All() {
		out[AxisMixin] = append(out[AxisMixin], m.Name)
	}
	for _, names := range out {
		slices.Sort(names)
	}
	return out
}

// Compare diffs what the corpus stamped against what eidos registers.
//
// stamped is keyed by the same axis constants and holds the classification
// names an annotation run produced. Names present in stamped but absent from
// the registry are ignored rather than reported: that is an eidos-side
// inconsistency, and pointing at it from here would blame the wrong repository
// for something the corpus cannot fix.
func Compare(stamped map[string][]string) Coverage {
	cov := Coverage{
		Stamped: make(map[string][]string, len(stamped)),
		Missing: make(map[string][]string, len(stamped)),
	}
	registered := Registered()

	for axis, names := range registered {
		seen := make(map[string]struct{}, len(stamped[axis]))
		for _, n := range stamped[axis] {
			seen[n] = struct{}{}
		}
		cov.Stamped[axis] = slices.Sorted(maps.Keys(seen))

		missing := make([]string, 0)
		for _, n := range names {
			if _, ok := seen[n]; !ok {
				missing = append(missing, n)
			}
		}
		cov.Missing[axis] = missing
	}
	return cov
}

// Complete reports whether every registered classification was stamped.
func (c Coverage) Complete() bool {
	for _, missing := range c.Missing {
		if len(missing) > 0 {
			return false
		}
	}
	return true
}

// String renders the gap, and the stamped set alongside it.
//
// Printing what *was* stamped is not padding. The gate runs eidos's annotator,
// so a gap has two possible causes — a fixture that is missing, or a fixture
// whose directive the annotator declined to read — and the stamped set is what
// tells them apart. Without it every failure reads as "write more fixtures",
// including the ones where the fixture already exists.
func (c Coverage) String() string {
	var b strings.Builder
	for _, axis := range []string{AxisDetector, AxisContract, AxisMixin} {
		missing, stamped := c.Missing[axis], c.Stamped[axis]
		fmt.Fprintf(&b, "%s: %d stamped, %d missing\n", axis, len(stamped), len(missing))
		if len(missing) > 0 {
			fmt.Fprintf(&b, "  missing: %s\n", strings.Join(missing, " "))
		}
		if len(stamped) > 0 {
			fmt.Fprintf(&b, "  stamped: %s\n", strings.Join(stamped, " "))
		}
	}
	return b.String()
}
