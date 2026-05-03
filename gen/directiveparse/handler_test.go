// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directiveparse_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/directiveparse"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Register and Get", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.NewRegistry()
		r.Register(directiveparse.Descriptor{Name: "errors", Description: "sentinel errors"})

		d, ok := r.Get("errors")
		testkit.True(t, ok, "must find registered descriptor")
		testkit.Equal(t, d.Name, "errors", "name must match")
		testkit.Equal(t, d.Description, "sentinel errors", "description must match")
	})

	t.Run("Get returns false for unknown", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.NewRegistry()
		_, ok := r.Get("nonexistent")
		testkit.False(t, ok, "must return false for unknown")
	})

	t.Run("IsKnown", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.NewRegistry()
		r.Register(directiveparse.Descriptor{Name: "errors"})
		testkit.True(t, r.IsKnown("errors"), "must be known")
		testkit.False(t, r.IsKnown("nonexistent"), "must not be known")
	})

	t.Run("Names returns sorted list", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.NewRegistry()
		r.Register(directiveparse.Descriptor{Name: "errors"})
		r.Register(directiveparse.Descriptor{Name: "concurrent"})
		r.Register(directiveparse.Descriptor{Name: "allocs"})

		testkit.Equal(t, r.Names(), []string{"allocs", "concurrent", "errors"}, "must be sorted")
	})

	t.Run("duplicate registration panics", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.NewRegistry()
		r.Register(directiveparse.Descriptor{Name: "errors"})

		testkit.Panics(t, func() {
			r.Register(directiveparse.Descriptor{Name: "errors"})
		}, "must panic on duplicate")
	})

	t.Run("empty registry has no names", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.NewRegistry()
		testkit.Len(t, r.Names(), 0, "empty registry")
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("known directives pass", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.NewRegistry()
		r.Register(directiveparse.Descriptor{Name: "errors"})
		r.Register(directiveparse.Descriptor{Name: "idempotent"})

		methods := []gen.MethodInfo{
			{Name: "Get", Directives: []gen.Directive{
				{Name: "errors", Args: []string{"ErrNotFound"}},
				{Name: "idempotent"},
			}},
		}
		errs := r.Validate(methods, nil)
		testkit.Len(t, errs, 0, "known directives must pass")
	})

	t.Run("unknown directive is error", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.NewRegistry()
		r.Register(directiveparse.Descriptor{Name: "errors"})

		methods := []gen.MethodInfo{
			{Name: "Get", Directives: []gen.Directive{
				{Name: "idempotnet"},
			}},
		}
		errs := r.Validate(methods, nil)
		testkit.Len(t, errs, 1, "unknown directive must error")
		testkit.Assert(t, errs[0].Error()).Contains("idempotnet", "must name the typo")
	})

	t.Run("experimental prefix warns instead of erroring", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.NewRegistry()

		var warnings []string
		methods := []gen.MethodInfo{
			{Name: "Get", Directives: []gen.Directive{
				{Name: "experimental:linearizable"},
			}},
		}
		errs := r.Validate(methods, func(msg string) { warnings = append(warnings, msg) })
		testkit.Len(t, errs, 0, "experimental must not error")
		testkit.Len(t, warnings, 1, "must produce warning")
		testkit.Assert(t, warnings[0]).Contains("experimental:linearizable", "must name directive")
	})

	t.Run("no directives passes", func(t *testing.T) {
		t.Parallel()
		r := directiveparse.NewRegistry()
		methods := []gen.MethodInfo{{Name: "Get"}}
		errs := r.Validate(methods, nil)
		testkit.Len(t, errs, 0, "no directives must pass")
	})
}
