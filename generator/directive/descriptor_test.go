// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

func TestDescriptor(t *testing.T) {
	t.Parallel()

	t.Run("New constructs a valid descriptor", func(t *testing.T) {
		t.Parallel()
		d := directive.New("conservative",
			directive.Describe("sum invariant"),
			directive.InCategory(directive.Mixin),
			directive.InPhase(directive.Phase3),
			directive.Arg("Field", directive.ArgIdent, directive.Required),
			directive.ConflictsWith("pure"),
		)
		testkit.Equal(t, d.Name, "conservative", "Name preserved")
		testkit.Equal(t, d.Category, directive.Mixin, "Category set")
		testkit.Len(t, d.Args, 1, "single arg")
		testkit.True(t, d.Args[0].Required, "Required option applied")
		testkit.Len(t, d.Conflicts, 1, "ConflictsWith applied")
	})

	t.Run("New panics on missing category", func(t *testing.T) {
		t.Parallel()
		assertPanic(t, func() {
			directive.New("x", directive.Describe("no category"))
		}, "missing-category panic")
	})

	t.Run("New panics when Multi is not on the last arg", func(t *testing.T) {
		t.Parallel()
		assertPanic(t, func() {
			directive.New("x",
				directive.InCategory(directive.Enrichment),
				directive.Arg("a", directive.ArgIdent, directive.Multi),
				directive.Arg("b", directive.ArgIdent),
			)
		}, "Multi-on-non-last panic")
	})

	t.Run("New panics on self-conflict", func(t *testing.T) {
		t.Parallel()
		assertPanic(t, func() {
			directive.New("x",
				directive.InCategory(directive.Mixin),
				directive.ConflictsWith("x"),
			)
		}, "self-conflict panic")
	})

	t.Run("New panics on ArgEnum without values", func(t *testing.T) {
		t.Parallel()
		assertPanic(t, func() {
			directive.New("x",
				directive.InCategory(directive.Enrichment),
				directive.Arg("a", directive.ArgEnum),
			)
		}, "enum-without-values panic")
	})

	t.Run("ValidateArgs accepts well-formed positional args", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		desc, ok := r.Get("errors")
		testkit.True(t, ok, "errors descriptor exists")
		testkit.Len(t, desc.ValidateArgs([]string{"ErrA", "ErrB"}, false), 0, "valid idents")
	})

	t.Run("ValidateArgs flags missing required arg", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		desc, _ := r.Get("timeout")
		errs := desc.ValidateArgs([]string{}, false)
		testkit.True(t, len(errs) > 0, "timeout requires duration arg")
	})

	t.Run("ValidateArgs returns no errors when Off", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		desc, _ := r.Get("timeout")
		testkit.Len(t, desc.ValidateArgs(nil, true), 0, "Off skips validation")
	})

	t.Run("ValidateArgs reports surplus args", func(t *testing.T) {
		t.Parallel()
		d := directive.New("x",
			directive.InCategory(directive.Enrichment),
			directive.Arg("a", directive.ArgIdent, directive.Required))
		errs := d.ValidateArgs([]string{"A", "Extra"}, false)
		testkit.True(t, len(errs) > 0, "extra arg flagged")
	})

	t.Run("ComposesWith and Experimental options take effect", func(t *testing.T) {
		t.Parallel()
		d := directive.New("x",
			directive.InCategory(directive.Mixin),
			directive.ComposesWith("y", "z"),
			directive.Experimental(),
		)
		testkit.Len(t, d.ComposesWith, 2, "ComposesWith populated")
		testkit.True(t, d.Experimental, "Experimental flag set")
	})
}

// assertPanic invokes fn and fails the test if fn doesn't panic.
func assertPanic(t *testing.T, fn func(), msg string) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic: %s", msg)
		}
	}()
	fn()
}
