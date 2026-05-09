// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"go/token"
	"sort"
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

// RenderMethodDirectives returns one source-form line per
// `//testkit:` directive attached to any method in `methods`,
// prefixed with the method name for audit-trail readability:
//
//	"Submit: //testkit:errors ErrNotFound ErrConflict  [stub: Fault<Sentinel>() helper per name]"
//	"Wrap:   //testkit:wrapped-via ErrInternal         [stub: Fault<Sentinel> helpers wrap via target]"
//	"Legacy: //testkit:deprecated Submit               [stub: tb.Logf in dispatch + // Deprecated: doc comment]"
//
// The method-name prefix is right-padded to align across all
// emitted lines so visual scanning matches the source. Each line
// is followed by a `[consumer: action, …]` annotation when the
// directive's [directive.Descriptor] declares Consumers — the
// reader sees, per directive, which generators it shapes and
// what each emits. Methods with no directives are silently
// skipped.
//
// Generators surface this in the file header so reviewers can
// audit which directives shaped the generated output without
// grepping the source.
func RenderMethodDirectives(methods []MethodInfo) []string {
	maxName := 0
	for _, m := range methods {
		for range m.Directives {
			if len(m.Name) > maxName {
				maxName = len(m.Name)
			}
		}
	}
	if maxName == 0 {
		return nil
	}
	reg := directive.DefaultRegistry()
	// Pre-compute the body (`name: //testkit:dir args`) for every
	// entry so we can right-pad to a uniform width before appending
	// the consumer annotation.
	type entry struct {
		body      string
		consumers []string
	}
	entries := make([]entry, 0)
	maxBody := 0
	for _, m := range methods {
		for _, d := range m.Directives {
			var b strings.Builder
			b.WriteString(m.Name)
			b.WriteByte(':')
			for i := len(m.Name); i < maxName; i++ {
				b.WriteByte(' ')
			}
			b.WriteByte(' ')
			b.WriteString(directivePrefix)
			b.WriteString(d.Name)
			for _, a := range d.Args {
				b.WriteByte(' ')
				b.WriteString(a)
			}
			body := b.String()
			if len(body) > maxBody {
				maxBody = len(body)
			}
			entries = append(entries, entry{body: body, consumers: consumerAnnotations(reg, d.Name)})
		}
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if len(e.consumers) == 0 {
			out = append(out, e.body)
			continue
		}
		var b strings.Builder
		b.WriteString(e.body)
		for i := len(e.body); i < maxBody; i++ {
			b.WriteByte(' ')
		}
		b.WriteString("  [")
		b.WriteString(strings.Join(e.consumers, ", "))
		b.WriteString("]")
		out = append(out, b.String())
	}
	return out
}

// consumerAnnotations renders the descriptor's Consumers map as a
// stable-ordered slice of "name: action" strings. Returns nil when
// the directive is unknown or has no consumer mappings — the
// caller drops the annotation suffix entirely in that case.
//
// Order: stub → suite → bench → model → other (alphabetical) —
// matches the audit reading order most reviewers want.
func consumerAnnotations(reg *directive.Registry, name string) []string {
	desc, ok := reg.Get(name)
	if !ok || len(desc.Consumers) == 0 {
		return nil
	}
	preferredOrder := []string{"stub", "suite", "bench", "model"}
	out := make([]string, 0, len(desc.Consumers))
	seen := make(map[string]bool, len(desc.Consumers))
	for _, k := range preferredOrder {
		if action, ok := desc.Consumers[k]; ok {
			out = append(out, k+": "+action)
			seen[k] = true
		}
	}
	rest := make([]string, 0)
	for k := range desc.Consumers {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		out = append(out, k+": "+desc.Consumers[k])
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
