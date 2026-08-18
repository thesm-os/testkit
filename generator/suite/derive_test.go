// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("every deriver is named", func(t *testing.T) {
		t.Parallel()
		for _, d := range suite.Registry() {
			testkit.NotEqual(t, d.Name(), suite.DeriverName(""),
				"an unnamed deriver cannot be attributed in refusals")
		}
	})

	t.Run("no name registers twice", func(t *testing.T) {
		t.Parallel()
		seen := map[suite.DeriverName]bool{}
		for _, d := range suite.Registry() {
			testkit.False(t, seen[d.Name()], "deriver "+string(d.Name())+" must register once")
			seen[d.Name()] = true
		}
	})
}
