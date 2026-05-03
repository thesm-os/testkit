// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directiveparse_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/directiveparse"
)

func TestDefaultRegistry(t *testing.T) {
	t.Parallel()

	t.Run("contains all known directives", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.DefaultRegistry()
		testkit.Len(t, r.Names(), 33, "must have 33 known directives")
	})

	t.Run("errors is known", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.DefaultRegistry()
		testkit.True(t, r.IsKnown("errors"), "errors must be known")
	})

	t.Run("typo is not known", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.DefaultRegistry()
		testkit.False(t, r.IsKnown("erors"), "typo must not be known")
	})
}
