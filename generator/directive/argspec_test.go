// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

func TestArgKind(t *testing.T) {
	t.Parallel()

	t.Run("String returns canonical kind name", func(t *testing.T) {
		t.Parallel()
		cases := map[directive.ArgKind]string{
			directive.ArgString:   "string",
			directive.ArgIdent:    "ident",
			directive.ArgInt:      "int",
			directive.ArgDuration: "duration",
			directive.ArgRange:    "range",
			directive.ArgEnum:     "enum",
			directive.ArgKey:      "key",
		}
		for k, want := range cases {
			testkit.Equal(t, k.String(), want, "ArgKind String")
		}
	})
}

func TestArgValidation(t *testing.T) {
	t.Parallel()

	// We exercise the per-kind validator indirectly via Descriptor.ValidateArgs
	// because validateArg itself is package-private. The cases below
	// each construct a one-arg descriptor of the relevant kind.

	t.Run("ArgIdent accepts Go identifiers", func(t *testing.T) {
		t.Parallel()
		d := directive.New("x", directive.InCategory(directive.Enrichment),
			directive.Arg("name", directive.ArgIdent, directive.Required))
		testkit.Len(t, d.ValidateArgs([]string{"User"}, false), 0, "valid ident")
		testkit.True(t, len(d.ValidateArgs([]string{"123-bad"}, false)) > 0, "invalid ident")
	})

	t.Run("ArgInt accepts integers only", func(t *testing.T) {
		t.Parallel()
		d := directive.New("x", directive.InCategory(directive.Enrichment),
			directive.Arg("n", directive.ArgInt, directive.Required))
		testkit.Len(t, d.ValidateArgs([]string{"42"}, false), 0, "valid int")
		testkit.True(t, len(d.ValidateArgs([]string{"forty"}, false)) > 0, "non-numeric")
	})

	t.Run("ArgDuration accepts Go durations", func(t *testing.T) {
		t.Parallel()
		d := directive.New("x", directive.InCategory(directive.Enrichment),
			directive.Arg("d", directive.ArgDuration, directive.Required))
		testkit.Len(t, d.ValidateArgs([]string{"100ms"}, false), 0, "valid duration")
		testkit.True(t, len(d.ValidateArgs([]string{"forever"}, false)) > 0, "invalid duration")
	})

	t.Run("ArgRange accepts min..max", func(t *testing.T) {
		t.Parallel()
		d := directive.New("x", directive.InCategory(directive.Mixin),
			directive.Arg("r", directive.ArgRange, directive.Required))
		testkit.Len(t, d.ValidateArgs([]string{"0..1"}, false), 0, "0..1")
		testkit.Len(t, d.ValidateArgs([]string{"-5..3.14"}, false), 0, "negatives + decimals")
		testkit.True(t, len(d.ValidateArgs([]string{"foo"}, false)) > 0, "non-range")
		testkit.True(t, len(d.ValidateArgs([]string{"a..b"}, false)) > 0, "non-numeric bounds")
	})

	t.Run("ArgEnum restricts to declared values", func(t *testing.T) {
		t.Parallel()
		d := directive.New("x", directive.InCategory(directive.Enrichment),
			directive.Arg("g", directive.ArgEnum, directive.Required, directive.OneOf("at-least-once", "at-most-once")))
		testkit.Len(t, d.ValidateArgs([]string{"at-least-once"}, false), 0, "valid value")
		testkit.True(t, len(d.ValidateArgs([]string{"exactly-once"}, false)) > 0, "rejects unlisted")
	})

	t.Run("Multi consumes remaining positional args", func(t *testing.T) {
		t.Parallel()
		d := directive.New("x", directive.InCategory(directive.Enrichment),
			directive.Arg("names", directive.ArgIdent, directive.Required, directive.Multi))
		testkit.Len(t, d.ValidateArgs([]string{"A", "B", "C"}, false), 0, "all valid")
	})
}
