// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/directive"
)

func TestValidateComposition(t *testing.T) {
	t.Parallel()

	t.Run("no directives has no issues", func(t *testing.T) {
		t.Parallel()
		issues := directive.ValidateComposition(nil)
		testkit.Len(t, issues, 0, "empty must pass")
	})

	t.Run("independent directives compose cleanly", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{
			{Name: "errors", Args: []string{"ErrNotFound"}},
			{Name: "idempotent"},
			{Name: "concurrent"},
		}
		issues := directive.ValidateComposition(dirs)
		testkit.Len(t, issues, 0, "independent directives must compose")
	})

	t.Run("pure conflicts with sideeffect", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{{Name: "pure"}, {Name: "sideeffect", Args: []string{"Get"}}}
		issues := directive.ValidateComposition(dirs)
		testkit.Len(t, issues, 1, "must detect conflict")
		testkit.Equal(t, issues[0].Kind, directive.Conflict, "must be conflict")
		testkit.Assert(t, issues[0].Message).Contains("pure", "must mention pure")
	})

	t.Run("pure conflicts with monotonic", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{{Name: "pure"}, {Name: "monotonic"}}
		issues := directive.ValidateComposition(dirs)
		testkit.Len(t, issues, 1, "must detect conflict")
		testkit.Equal(t, issues[0].Kind, directive.Conflict, "must be conflict")
	})

	t.Run("cacheable conflicts with sideeffect", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{{Name: "cacheable"}, {Name: "sideeffect", Args: []string{"Get"}}}
		issues := directive.ValidateComposition(dirs)
		testkit.Len(t, issues, 1, "must detect conflict")
		testkit.Equal(t, issues[0].Kind, directive.Conflict, "must be conflict")
	})

	t.Run("cacheable conflicts with monotonic", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{{Name: "cacheable"}, {Name: "monotonic"}}
		issues := directive.ValidateComposition(dirs)
		testkit.Len(t, issues, 1, "must detect conflict")
	})

	t.Run("concurrent conflicts with concurrent-readers", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{{Name: "concurrent"}, {Name: "concurrent-readers"}}
		issues := directive.ValidateComposition(dirs)
		testkit.Len(t, issues, 1, "must detect conflict")
		testkit.Equal(t, issues[0].Kind, directive.Conflict, "must be conflict")
	})

	t.Run("retry-succeeds-on requires retryable", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{{Name: "retry-succeeds-on", Args: []string{"3"}}}
		issues := directive.ValidateComposition(dirs)
		testkit.Len(t, issues, 1, "must detect missing required")
		testkit.Equal(t, issues[0].Kind, directive.MissingRequired, "must be missing-required")
		testkit.Assert(t, issues[0].Message).Contains("retryable", "must name required directive")
	})

	t.Run("retry-succeeds-on with retryable is fine", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{{Name: "retryable"}, {Name: "retry-succeeds-on", Args: []string{"3"}}}
		issues := directive.ValidateComposition(dirs)
		testkit.Len(t, issues, 0, "must pass with required pair present")
	})

	t.Run("pure with idempotent is redundant", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{{Name: "pure"}, {Name: "idempotent"}}
		issues := directive.ValidateComposition(dirs)
		testkit.Len(t, issues, 1, "must detect redundancy")
		testkit.Equal(t, issues[0].Kind, directive.Redundant, "must be redundant")
		testkit.Assert(t, issues[0].Message).Contains("implies", "must say implies")
	})

	t.Run("cacheable with pure is redundant", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{{Name: "cacheable"}, {Name: "pure"}}
		issues := directive.ValidateComposition(dirs)
		testkit.Len(t, issues, 1, "must detect redundancy")
		testkit.Equal(t, issues[0].Kind, directive.Redundant, "must be redundant")
	})

	t.Run("cacheable with idempotent is redundant", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{{Name: "cacheable"}, {Name: "idempotent"}}
		issues := directive.ValidateComposition(dirs)
		testkit.Len(t, issues, 1, "must detect redundancy")
	})

	t.Run("multiple issues detected at once", func(t *testing.T) {
		t.Parallel()
		dirs := []gen.Directive{
			{Name: "pure"},
			{Name: "idempotent"},
			{Name: "sideeffect", Args: []string{"Get"}},
		}
		issues := directive.ValidateComposition(dirs)
		testkit.True(t, len(issues) >= 2, "must detect both conflict and redundancy")
	})
}
