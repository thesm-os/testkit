// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"go/token"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("DefaultRegistry has descriptors in every category", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		testkit.True(t, len(r.Names()) >= 30, "expect ≥ 30 known directives")

		for _, c := range []directive.Category{
			directive.SignatureHint, directive.Mixin,
			directive.Enrichment, directive.Documentation,
		} {
			testkit.True(t, len(r.DescriptorsIn(c)) > 0, "category populated: "+c.String())
		}
	})

	t.Run("Validate accepts well-formed args", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		pos := token.Position{Filename: "x.go"}
		errs := r.Validate([]directive.Directive{
			{Name: "errors", Args: []string{"ErrNotFound", "ErrBadInput"}},
			{Name: "timeout", Args: []string{"1s"}},
			{Name: "bounded", Args: []string{"0..1"}},
		}, "M", pos, nil)
		testkit.Len(t, errs, 0, "well-formed directives validate")
	})

	t.Run("Validate rejects malformed args by kind", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		pos := token.Position{Filename: "x.go"}
		cases := []struct {
			name string
			d    directive.Directive
		}{
			{"errors with non-ident arg", directive.Directive{Name: "errors", Args: []string{"123-bad"}}},
			{"timeout with bad duration", directive.Directive{Name: "timeout", Args: []string{"not-a-duration"}}},
			{"bounded with malformed range", directive.Directive{Name: "bounded", Args: []string{"foo"}}},
		}
		for _, tc := range cases {
			errs := r.Validate([]directive.Directive{tc.d}, "M", pos, nil)
			testkit.True(t, len(errs) > 0, "must reject: "+tc.name)
		}
	})

	t.Run("Validate flags unknown directives", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		errs := r.Validate([]directive.Directive{{Name: "totally-fake"}}, "M", token.Position{}, nil)
		testkit.True(t, len(errs) > 0, "unknown directive errors")
	})

	t.Run("Off directives skip arg validation", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		errs := r.Validate([]directive.Directive{{Name: "timeout", Off: true}}, "M", token.Position{}, nil)
		testkit.Len(t, errs, 0, "Off=true skips arg checks")
	})

	t.Run("Descriptors returns the full sorted set", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		all := r.Descriptors()
		testkit.True(t, len(all) >= 30, "≥ 30 descriptors")
		for i := 1; i < len(all); i++ {
			testkit.True(t, all[i-1].Name < all[i].Name,
				"sorted: "+all[i-1].Name+" before "+all[i].Name)
		}
	})

	t.Run("MustRegister panics on duplicate descriptor", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		r.MustRegister(directive.New("x", directive.InCategory(directive.Mixin)))
		defer func() {
			testkit.True(t, recover() != nil, "duplicate must panic")
		}()
		r.MustRegister(directive.New("x", directive.InCategory(directive.Mixin)))
	})

	t.Run("experimental: prefix routes through warn callback", func(t *testing.T) {
		t.Parallel()
		r := directive.DefaultRegistry()
		warned := 0
		errs := r.Validate(
			[]directive.Directive{{Name: "experimental:custom-thing"}},
			"M", token.Position{},
			func(string) { warned++ },
		)
		testkit.Len(t, errs, 0, "experimental never errors")
		testkit.Equal(t, warned, 1, "warn called once")
	})
}
