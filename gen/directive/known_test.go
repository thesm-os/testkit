// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/directive"
)

func TestDefaultRegistry(t *testing.T) {
	t.Parallel()

	t.Run("contains all known directives", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		testkit.Len(t, r.Names(), 32, "must have 32 known directives")
	})

	t.Run("errors is known", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		testkit.True(t, r.IsKnown("errors"), "errors must be known")
	})

	t.Run("typo is not known", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		testkit.False(t, r.IsKnown("erors"), "typo must not be known")
	})
}
