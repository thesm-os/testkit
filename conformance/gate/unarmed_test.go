// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/gate"
)

// consumerText concatenates a fixture's hand-written test files — every
// non-generated *_test.go under its directory tree, which is where a corpus
// consumer arms a generated option. Generated files are excluded so a
// falsification companion mentioning an option does not count as arming it.
func consumerText(t *testing.T, dir string) string {
	t.Helper()
	var sb strings.Builder
	base := filepath.Join(corpusRoot, "corpus", dir)
	err := filepath.WalkDir(base,
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := d.Name()
			if d.IsDir() || !strings.HasSuffix(name, "_test.go") || strings.Contains(name, ".gen") {
				return nil
			}
			// The corpus is this repository's own checkout, not untrusted
			// input; the walk cannot race a symlink nobody plants.
			body, rErr := os.ReadFile(path) //nolint:gosec // see above
			if rErr != nil {
				return fmt.Errorf("read %s: %w", path, rErr)
			}
			sb.Write(body)
			return nil
		})
	testkit.NoError(t, err, "the fixture's consumer tests are readable")
	return sb.String()
}

// TestEveryDoorIsArmedOrArgued is the unarmed-door census: a guarded law, a
// clocked law, or an undeclared optional role is a visible skip unless some
// consumer arms it or the register argues why not — and an argued row for a
// door that is now armed is a stale excuse. No counts anywhere: the door
// set derives from the emission, the arming from the consumers' own calls,
// and every failure names its fixture and law.
func TestEveryDoorIsArmedOrArgued(t *testing.T) {
	t.Parallel()

	census, err := censusOnce()
	emitted := census.Emitted
	testkit.NoError(t, err, "the emission census runs")

	seen := map[string]bool{}
	verdict := func(key, armed string, isArmed bool) {
		seen[key] = true
		reason, argued := gate.UnarmedDoors[key]
		switch {
		case isArmed && argued:
			t.Errorf("%s is armed by a consumer (%s) and still registered — "+
				"delete the stale row: %s", key, armed, reason)
		case !isArmed && !argued:
			t.Errorf("%s is neither armed by a consumer (no %s in the "+
				"fixture's tests) nor argued in gate.UnarmedDoors — a guarded "+
				"law nobody arms is a visible skip", key, armed)
		}
	}

	for _, e := range emitted {
		if e.Dir == "" {
			continue
		}
		tests := consumerText(t, e.Dir)
		for law, doors := range e.Doors {
			for _, door := range doors {
				// The option exports the config field: first letter up, the
				// same spelling the supplied-option surface generates.
				call := e.IfaceName + "Model" + strings.ToUpper(door[:1]) + door[1:] + "("
				verdict(e.Dir+"/"+law+"."+door, call, strings.Contains(tests, call))
			}
		}
		for _, law := range e.Clocked {
			call := e.IfaceName + "ModelClocked("
			verdict(e.Dir+"/"+law+".clock", call, strings.Contains(tests, call))
		}
		for law, roles := range e.Unarmed {
			for _, role := range roles {
				// An undeclared optional role has no option to arm — the
				// declaration itself is the arming — so a row is always owed.
				verdict(e.Dir+"/"+law+"."+role, "redeliver= on the directive", false)
			}
		}
	}

	for key := range gate.UnarmedDoors {
		testkit.True(t, seen[key],
			key+" is registered but the emission produced no such door — the row outlived its debt")
	}
}

// TestStampedMissIdentityReachesTheSequences holds the declaration's routed
// miss sentinel to the sequences that compare it: a fixture whose stamp the
// derived oracle consumes must also arm the identity on its reader actions,
// or the comparison silently regressed to presence.
func TestStampedMissIdentityReachesTheSequences(t *testing.T) {
	t.Parallel()

	census, err := censusOnce()
	emitted := census.Emitted
	testkit.NoError(t, err, "the emission census runs")

	for _, e := range emitted {
		if e.SentinelStamped {
			testkit.True(t, e.SentinelArmed,
				e.Fixture+" stamps a miss sentinel the oracle routes, and no sequence carries it")
		}
	}
}
