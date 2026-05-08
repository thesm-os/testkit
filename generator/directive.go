// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"go/token"
	"strings"

	"go.thesmos.sh/testkit/generator/directive"
)

// directivePrefix marks a comment line as a testkit directive.
const directivePrefix = "//testkit:"

// parseDirectivesFromDoc extracts all //testkit: directives from a doc
// comment string. The position is approximate — it reflects the
// declaration the doc comment is attached to, not the directive line
// itself, since go/ast doesn't preserve per-line positions in
// extracted doc text.
//
// Two syntactic forms are accepted:
//
//	//testkit:errors ErrNotFound ErrBadInput      // standalone
//	//testkit:directive atomic idempotent ctx     // bundle
//	//testkit:directive timeout=1s writer=off     // bundle with values
//
// In bundle form each whitespace-separated token is its own directive;
// "name=value" carries a single arg (or comma-separated multi-args via
// "name=v1,v2"); "name=off" sets the directive's Off flag for opt-out
// semantics.
//
// Token recognition delegates to [directive.ParseLine].
func parseDirectivesFromDoc(doc string, pos token.Position) []directive.Directive {
	if doc == "" {
		return nil
	}
	var out []directive.Directive
	for line := range strings.SplitSeq(doc, "\n") {
		// Doc.Text() strips "// " prefixes already; raw scanning may not.
		// Accept both forms.
		raw := strings.TrimPrefix(strings.TrimSpace(line), "//")
		raw = strings.TrimSpace(raw)
		if !strings.HasPrefix(raw, "testkit:") {
			continue
		}
		body := strings.TrimPrefix(raw, "testkit:")
		for _, t := range directive.ParseLine(body) {
			out = append(out, directive.Directive{
				Name: t.Name,
				Args: t.Args,
				Off:  t.Off,
				Pos:  pos,
			})
		}
	}
	return out
}

// RenderPackageDirectives returns the package's `//testkit:` directives
// (filtered to names if non-empty) as source lines, e.g.
// "//testkit:sentinel-no-overlap-with example.com/x". Generators surface
// these in the generated file's header doc comment so the reader sees
// the inputs without grepping the source package.
//
// Pass nil/empty names to surface every package directive; pass a
// filter to restrict output to directives the generator actually
// consumes.
func RenderPackageDirectives(pkg *Package, names ...string) []string {
	keep := func(string) bool { return true }
	if len(names) > 0 {
		nameSet := make(map[string]struct{}, len(names))
		for _, n := range names {
			nameSet[n] = struct{}{}
		}
		keep = func(n string) bool { _, ok := nameSet[n]; return ok }
	}
	var out []string
	for _, d := range pkg.PackageDirectives() {
		if !keep(d.Name) {
			continue
		}
		var b strings.Builder
		b.WriteString(directivePrefix)
		b.WriteString(d.Name)
		for _, a := range d.Args {
			b.WriteByte(' ')
			b.WriteString(a)
		}
		out = append(out, b.String())
	}
	return out
}
