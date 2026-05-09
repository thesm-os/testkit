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

	t.Run("String falls back to unknown for unrecognised kinds", func(t *testing.T) {
		t.Parallel()
		// Out-of-range value triggers the default branch.
		testkit.Equal(t, directive.ArgKind(99).String(), "unknown",
			"unrecognised kind → unknown")
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

	t.Run("ArgString rejects empty values", func(t *testing.T) {
		t.Parallel()
		d := directive.New("x", directive.InCategory(directive.Enrichment),
			directive.Arg("s", directive.ArgString, directive.Required))
		errs := d.ValidateArgs([]string{""}, false)
		testkit.True(t, len(errs) > 0, "empty string rejected")
		testkit.Assert(t, errs[0].Error()).Contains("empty string", "diagnostic")
	})

	t.Run("ArgRange flags invalid upper bound", func(t *testing.T) {
		t.Parallel()
		// lo parses OK, hi fails — exercises the second ParseFloat branch.
		d := directive.New("x", directive.InCategory(directive.Mixin),
			directive.Arg("r", directive.ArgRange, directive.Required))
		errs := d.ValidateArgs([]string{"0.0..bad"}, false)
		testkit.True(t, len(errs) > 0, "non-numeric upper rejected")
		testkit.Assert(t, errs[0].Error()).Contains("upper bound", "names the bad side")
	})

	t.Run("ArgKey behaves like ArgIdent (Go-identifier rules)", func(t *testing.T) {
		t.Parallel()
		d := directive.New("x", directive.InCategory(directive.Enrichment),
			directive.Arg("k", directive.ArgKey, directive.Required))
		testkit.Len(t, d.ValidateArgs([]string{"User"}, false), 0, "valid ident")
		testkit.True(t, len(d.ValidateArgs([]string{"User-name"}, false)) > 0,
			"hyphen rejected (isGoIdent default branch)")
	})

	t.Run("ArgIdent rejects empty values via isGoIdent", func(t *testing.T) {
		t.Parallel()
		// Surplus arg slot with empty value passes through to
		// isGoIdent("") — exercises the empty-string early-return.
		d := directive.New("x", directive.InCategory(directive.Enrichment),
			directive.Arg("a", directive.ArgIdent, directive.Required, directive.Multi))
		errs := d.ValidateArgs([]string{""}, false)
		testkit.True(t, len(errs) > 0, "empty ident rejected")
	})
}
