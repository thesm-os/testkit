// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

func TestBodyKinds(t *testing.T) {
	t.Parallel()

	t.Run("unique", func(t *testing.T) {
		t.Parallel()
		seen := map[projection.BodyKind]bool{}
		for _, k := range projection.BodyKinds() {
			testkit.False(t, seen[k], "body kind "+string(k)+" must register once")
			seen[k] = true
		}
	})

	t.Run("dispatch-prefixed", func(t *testing.T) {
		t.Parallel()
		for _, k := range projection.BodyKinds() {
			testkit.HasPrefix(t, string(k), projection.BodyKindPrefix, "kinds are template names in the dispatch namespace")
		}
	})

	t.Run("within the design budget", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(projection.BodyKinds()) <= 15,
			"a new body shape is a design event, not a workaround — the budget is the guard")
	})
}

// simKindCase ties one sim kind to the runtime segment it must equal.
type simKindCase struct {
	name string
	kind projection.SimKind
	seg  string
}

func (c simKindCase) Name() string { return c.name }

func TestSimKindsAreTheRuntimeVocabulary(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []simKindCase{
		{"recovery", projection.SimRecovery, suite.SegRecovery},
		{"crash", projection.SimCrash, suite.SegCrash},
		{"fault", projection.SimFault, suite.SegFault},
	}, func(t *testing.T, tc simKindCase) {
		testkit.Equal(t, string(tc.kind), tc.seg,
			"the sim vocabulary has one home; a drifted spelling mints IDs the runtime refuses")
	})
}
